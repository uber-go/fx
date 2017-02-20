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
	"github.com/pkg/errors"
)

var (
	errParamType   = errors.New("graph injection must be done through a pointer or a function")
	errReturnCount = errors.New("constructor function must return exactly one value")
	errReturnKind  = errors.New("constructor return type must be a pointer")
	errArgKind     = errors.New("constructor arguments must be pointers")
	errCycle       = errors.New("cycle dependencies detected")
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
