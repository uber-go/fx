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
func New() *Graph {
	return &Graph{
		nodes: make(map[interface{}]graphNode),
	}
}

// Graph facilitates automated dependency resolution
type Graph struct {
	sync.Mutex

	nodes map[interface{}]graphNode
}

// Register an object in the dependency graph
//
// Provided argument must be a function that returns exactly one pointer argument
// All arguments to the function must be pointers
func (g *Graph) Register(c interface{}) error {
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
func (g *Graph) Resolve(obj interface{}) (err error) {
	g.Lock()
	defer g.Unlock()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during Resolve %v", r)
		}
	}()

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
func (g *Graph) ResolveAll(objs ...interface{}) error {
	for _, o := range objs {
		if err := g.Resolve(o); err != nil {
			return err
		}
	}
	return nil
}

// RegisterAll registers all the provided args in the dependency graph
func (g *Graph) RegisterAll(cs ...interface{}) error {
	for _, c := range cs {
		if err := g.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// Reset the graph by removing all the registered nodes
func (g *Graph) Reset() {
	g.Lock()
	defer g.Unlock()

	g.nodes = make(map[interface{}]graphNode)
}

// String representation of the entire graph
func (g *Graph) String() string {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "{nodes:")
	for key, reg := range g.nodes {
		fmt.Fprintln(b, key, "->", reg)
	}
	fmt.Fprintln(b, "}")
	return b.String()
}

func (g *Graph) registerObject(o interface{}) error {
	otype := reflect.TypeOf(o)

	n := objNode{
		obj: o,
		node: node{
			objType: otype,
			cached:  true,
		},
	}

	g.nodes[otype] = &n
	return nil
}

func (g *Graph) registerConstructor(c interface{}) error {
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

	// object needs to be part of the graph to properly detect cycles
	if cycleErr := g.detectCycles(&n); cycleErr != nil {
		// if the cycle was detected delete from the graph
		delete(g.nodes, objType)
		return errors.Wrapf(cycleErr, "unable to register %v", objType)
	}

	return nil
}

// When a new constructor is being inserted, detect any present cycles
func (g *Graph) detectCycles(n *funcNode) error {
	l := []string{}
	return g.recursiveDetectCycles(n, l)
}

// DFS and tracking if same node is visited twice
func (g *Graph) recursiveDetectCycles(n graphNode, l []string) error {
	for _, el := range l {
		if n.id() == el {
			b := &bytes.Buffer{}
			for _, curr := range l {
				fmt.Fprint(b, curr, " -> ")
			}
			fmt.Fprint(b, n.id())
			return fmt.Errorf("detected cycle %s", b.String())
		}
	}

	l = append(l, n.id())

	for _, dep := range n.dependencies() {
		if node, ok := g.nodes[dep]; ok {
			if err := g.recursiveDetectCycles(node, l); err != nil {
				return err
			}
		}
	}

	return nil
}
