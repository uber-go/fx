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
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type graphNode interface {
	// Return value of the object
	value(g *graph) (reflect.Value, error)

	// Other things that need to be present before this object can be created
	dependencies() []interface{}

	// unique identification per node
	//
	// TODO(glib): GFM-396
	// consider using a custom type to identify objects, rather than a string
	// type id struct { reflect.Type, string name, } or something of the sort
	id() string
}

type node struct {
	objType     reflect.Type
	cached      bool
	cachedValue reflect.Value
}

func (n node) id() string {
	// in the future, more than just the type of node is going to be required
	// for instance, when multiple types are allowed with different names
	//
	// TODO(glib): GFM-396
	// Type.String() is not guaranteed to be unique and can return the same value
	// for structs with the same name in a different package.
	return n.objType.String()
}

type objNode struct {
	node

	obj interface{}
}

// Return the earlier provided instance
func (n *objNode) value(g *graph) (reflect.Value, error) {
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
	node

	constructor interface{}
	deps        []interface{}
}

// Call the function and return the result
func (n *funcNode) value(g *graph) (reflect.Value, error) {
	if n.cached {
		return n.cachedValue, nil
	}

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

	cv := reflect.ValueOf(n.constructor)
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
		"(function) id: %s, deps: %v, type: %v, constructor: %v, cached: %v, cachedValue: %v",
		n.id(), n.deps, n.objType, n.constructor, n.cached, n.cachedValue,
	)
}
