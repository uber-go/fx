// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dig

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

var (
	errParamType   = errors.New("Graph injection must be done through a pointer or a function")
	errReturnCount = errors.New("Constructor function must return exactly one value")
	errReturnKind  = errors.New("Constructor return type must be a pointer")
	errArgKind     = errors.New("Constructor arguments must be pointers")
)

// New returns a new Dependency Injection Graph
func New() Graph {
	return newGraph()
}

// Graph facilitates basic dependency injection
//
// TODO: Inject functions should take options to control resolve behaviour
// for example, at the moment everything is a singleton and we need an option
// to initialize a new instance on every resolve
type Graph interface {
	// Inject into the dependency graph
	// Parameter must be a pointer or a constructor function returning exactly one pointer
	Inject(interface{}) error

	// Same as calling .Inject for each provided vararg
	// Returns the first error encountered
	InjectAll(...interface{}) error

	// Resolve the dependencies of the object and populate the pointer value
	Resolve(interface{}) error
}

// Inject an object in the dependency graph
//
// Provided argument must be a function that returns exactly one pointer argument
// All arguments to the function must be pointers
func (g *graph) Inject(c interface{}) error {
	ctype := reflect.TypeOf(c)

	switch ctype.Kind() {
	case reflect.Func:
		if ctype.NumOut() != 1 {
			return errReturnCount
		}

		objType := ctype.Out(0)
		if objType.Kind() != reflect.Ptr && objType.Kind() != reflect.Interface {
			return errReturnKind
		}

		return g.injectConstructor(c)
	case reflect.Ptr:
		// validation for pointers?
		return g.injectObject(c)
	default:
		return errParamType
	}
}

// Resolve all of the dependencies of the provided class
//
// Provided object must be a pointer
// Any dependencies of the object will receive constructor calls, or be initialized (once)
// Constructor with return value *object will be called
//
// TODO(glib): catch any and all panics from this method, as there is a lot of reflect going on
func (g *graph) Resolve(obj interface{}) error {
	objType := reflect.TypeOf(obj)
	if objType.Kind() != reflect.Ptr {
		return fmt.Errorf("can not resolve non-pointer object of type %v", objType)
	}

	objElemType := reflect.TypeOf(obj).Elem()
	objVal := reflect.ValueOf(obj)

	// check if the type has been nodes
	n, ok := g.nodes[objElemType]
	if !ok {
		return fmt.Errorf("type %v is not registered", objType)
	}

	// check that all the dependencies have nodes present in the graph
	// doesn't mean everything will go smoothly during resolve, but it
	// drastically increases the chances that we're not missing something
	for _, node := range g.nodes {
		for _, dep := range node.dependencies() {
			// check that the dependency has been nodes
			_, ok := g.nodes[dep]
			if !ok {
				return fmt.Errorf("%v dependency of type %v is not registered", objElemType, dep)
			}
		}
	}

	v, err := n.value(g)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve %v", objType)
	}

	// set the pointer value of the provided object to the instance pointer
	objVal.Elem().Set(v)

	return nil
}

type object interface {
	// Return value of the object
	value(g *graph) (reflect.Value, error)

	// Other things that need to be present before this object can be created
	dependencies() []interface{}
}

type graph struct {
	fmt.Stringer

	nodes map[interface{}]object
}

func newGraph() *graph {
	return &graph{
		nodes: make(map[interface{}]object),
	}
}

func (g graph) String() string {
	var b bytes.Buffer
	for key, reg := range g.nodes {
		b.WriteString(fmt.Sprintf("%v -> %v\n", key, reg))
	}
	return fmt.Sprintf("{nodes:\n%v}", b.String())
}

type node struct {
	fmt.Stringer

	obj         interface{}
	objType     reflect.Type
	cached      bool
	cachedValue reflect.Value
}

// Return the earlier provided instance
func (n node) value(g *graph) (reflect.Value, error) {
	return reflect.ValueOf(n.obj), nil
}

func (n node) dependencies() []interface{} {
	return nil
}

func (n node) String() string {
	return fmt.Sprintf(
		"(object) obj: %v, deps: nil, cached: %v",
		n.objType,
		n.cached,
	)
}

type funcNode struct {
	node
	fmt.Stringer

	constructor interface{}
	deps        []interface{}
}

// Call the function and return the result
func (n *funcNode) value(g *graph) (reflect.Value, error) {
	if n.cached {
		return n.cachedValue, nil
	}

	cv := reflect.ValueOf(n.constructor)
	ct := reflect.TypeOf(n.constructor)

	args := make([]reflect.Value, ct.NumIn(), ct.NumIn())
	for idx := range args {
		arg := ct.In(idx)
		node, ok := g.nodes[arg]
		if ok {
			v, err := node.value(g)
			if err != nil {
				return reflect.Zero(n.objType), errors.Wrap(err, "dependency resolution failed")
			}
			args[idx] = v
		}
	}

	v := cv.Call(args)[0]
	n.cached = true
	n.cachedValue = v

	return v, nil
}

func (n funcNode) dependencies() []interface{} {
	return n.deps
}

func (n funcNode) String() string {
	return fmt.Sprintf(
		"(function) deps: %v, type: %v, constructor: %v, cached: %v",
		n.deps,
		n.objType,
		n.constructor,
		n.cached,
	)
}

func (g *graph) injectObject(o interface{}) error {
	otype := reflect.TypeOf(o)

	n := node{
		obj:     o,
		objType: otype,
		cached:  true,
	}

	g.nodes[otype] = &n
	return nil
}

func (g *graph) injectConstructor(c interface{}) error {
	ctype := reflect.TypeOf(c)
	objType := ctype.Out(0)

	argc := ctype.NumIn()
	n := funcNode{
		deps:        make([]interface{}, argc),
		constructor: c,
		node: node{
			objType: objType,
		},
	}
	for i := 0; i < argc; i++ {
		arg := ctype.In(i)
		if arg.Kind() != reflect.Ptr && arg.Kind() != reflect.Interface {
			return errArgKind
		}

		n.deps[i] = arg
	}

	g.nodes[objType] = &n
	return nil
}

// InjectAll registers all the provided args in the dependency graph
func (g *graph) InjectAll(cs ...interface{}) error {
	for _, c := range cs {
		err := g.Inject(c)
		if err != nil {
			return err
		}
	}
	return nil
}
