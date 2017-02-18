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
	"sync"

	"github.com/pkg/errors"
)

var (
	errParamType   = errors.New("graph injection must be done through a pointer or a function")
	errReturnCount = errors.New("constructor function must return exactly one value")
	errReturnKind  = errors.New("constructor return type must be a pointer")
	errArgKind     = errors.New("constructor arguments must be pointers")
)

// New returns a new Dependency Injection Graph
func New() Graph {
	return newGraph()
}

// Graph facilitates basic dependency injection
//
// TODO: Register functions should take options to control resolve behaviour
// for example, at the moment everything is a singleton and we need an option
// to initialize a new instance on every resolve
type Graph interface {
	// Register into the dependency graph
	// Parameter must be a pointer or a constructor function returning exactly one pointer
	Register(interface{}) error

	// Same as calling .Register for each provided vararg
	// Returns the first error encountered
	RegisterAll(...interface{}) error

	// Resolve the dependencies of the object and populate the pointer value
	Resolve(interface{}) error

	// ResolveAll the passed in pointers through the dependency graph
	// Returns the first error encountered
	ResolveAll(...interface{}) error

	// Reset the graph by removing all the registered nodes
	Reset()
}

// Register an object in the dependency graph
//
// Provided argument must be a function that returns exactly one pointer argument
// All arguments to the function must be pointers
func (g *graph) Register(c interface{}) error {
	g.Lock()
	defer g.Unlock()

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

		return g.registerConstructor(c)
	case reflect.Ptr:
		return g.registerObject(c)
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
	g.Lock()
	defer g.Unlock()

	objType := reflect.TypeOf(obj)
	if objType.Kind() != reflect.Ptr {
		return fmt.Errorf("can not resolve non-pointer object of type %v", objType)
	}

	objElemType := reflect.TypeOf(obj).Elem()
	objVal := reflect.ValueOf(obj)

	// check if the type is a registered objNode
	n, ok := g.nodes[objElemType]
	if !ok {
		return fmt.Errorf("type %v is not registered", objType)
	}

	v, err := n.value(g)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve %v", objType)
	}

	// set the pointer value of the provided object to the instance pointer
	objVal.Elem().Set(v)

	return nil
}

// ResolveAll the dependencies of each provided object
// Returns the first error encountered
func (g *graph) ResolveAll(objs ...interface{}) error {
	for _, o := range objs {
		if err := g.Resolve(o); err != nil {
			return err
		}
	}
	return nil
}

// RegisterAll registers all the provided args in the dependency graph
func (g *graph) RegisterAll(cs ...interface{}) error {
	for _, c := range cs {
		if err := g.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// Reset the graph by removing all the registered nodes
func (g *graph) Reset() {
	g.Lock()
	defer g.Unlock()

	defaultGraph.nodes = make(map[interface{}]object)
}

type object interface {
	// Return value of the object
	value(g *graph) (reflect.Value, error)

	// Other things that need to be present before this object can be created
	dependencies() []interface{}
}

type graph struct {
	fmt.Stringer
	sync.Mutex

	nodes map[interface{}]object
}

func newGraph() *graph {
	return &graph{
		nodes: make(map[interface{}]object),
	}
}

func (g *graph) String() string {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "{nodes:")
	for key, reg := range g.nodes {
		fmt.Fprintln(b, key, " -> ", reg)
	}
	fmt.Fprintln(b, "}")
	return b.String()
}

type objNode struct {
	fmt.Stringer

	obj         interface{}
	objType     reflect.Type
	cached      bool
	cachedValue reflect.Value
}

// Return the earlier provided instance
func (n objNode) value(g *graph) (reflect.Value, error) {
	return reflect.ValueOf(n.obj), nil
}

func (n objNode) dependencies() []interface{} {
	return nil
}

func (n objNode) String() string {
	return fmt.Sprintf(
		"(object) obj: %v, deps: nil, cached: %v, cachedValue: %v",
		n.objType,
		n.cached,
		n.cachedValue,
	)
}

type funcNode struct {
	objNode
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

	// check that all the dependencies have nodes present in the graph
	// doesn't mean everything will go smoothly during resolve, but it
	// drastically increases the chances that we're not missing something
	for _, node := range g.nodes {
		for _, dep := range node.dependencies() {
			// check that the dependency is a registered objNode
			if _, ok := g.nodes[dep]; !ok {
				err := fmt.Errorf("%v dependency of type %v is not registered", ct, dep)
				return reflect.Zero(ct), err
			}
		}
	}

	args := make([]reflect.Value, ct.NumIn(), ct.NumIn())
	for idx := range args {
		arg := ct.In(idx)
		if node, ok := g.nodes[arg]; ok {
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
		"(function) deps: %v, type: %v, constructor: %v, cached: %v, cachedValue: %v",
		n.deps,
		n.objType,
		n.constructor,
		n.cached,
		n.cachedValue,
	)
}

func (g *graph) registerObject(o interface{}) error {
	otype := reflect.TypeOf(o)

	n := objNode{
		obj:     o,
		objType: otype,
		cached:  true,
	}

	g.nodes[otype] = &n
	return nil
}

func (g *graph) registerConstructor(c interface{}) error {
	ctype := reflect.TypeOf(c)
	objType := ctype.Out(0)

	argc := ctype.NumIn()
	n := funcNode{
		deps:        make([]interface{}, argc),
		constructor: c,
		objNode: objNode{
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
