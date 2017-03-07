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

// Default graph is used for all the top-level calls
var defaultGraph = New()

// DefaultGraph returns the graph used by top-level calls
func DefaultGraph() *Graph {
	return defaultGraph
}

// Register into the default graph
func Register(i interface{}) error {
	return defaultGraph.Register(i)
}

// MustRegister calls Register and panics if an error is encountered
func MustRegister(i interface{}) {
	defaultGraph.MustRegister(i)
}

// RegisterAll into the default graph
func RegisterAll(is ...interface{}) error {
	return defaultGraph.RegisterAll(is...)
}

// MustRegisterAll into the default graph
func MustRegisterAll(is ...interface{}) {
	defaultGraph.MustRegisterAll(is...)
}

// Resolve an object through the default graph
func Resolve(i interface{}) error {
	return defaultGraph.Resolve(i)
}

// MustResolve through the default graph
func MustResolve(i interface{}) {
	defaultGraph.MustResolve(i)
}

// ResolveAll through the default graph
func ResolveAll(is ...interface{}) error {
	return defaultGraph.ResolveAll(is...)
}

// MustResolveAll on the default graph
func MustResolveAll(is ...interface{}) {
	defaultGraph.MustResolveAll(is...)
}

// Reset the default graph
func Reset() {
	defaultGraph.Reset()
}
