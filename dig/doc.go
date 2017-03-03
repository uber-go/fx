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

// Package dig is the Dependency Injection Graph.
//
// package dig provides an opinionated way of resolving object dependencies.
// There are two sides of dig:
// Register and Resolve.
//
// Register
//
// Register adds an object, or a constructor of an object to the graph.
//
// There are two ways to register an object:
//
// • Register a pointer to an existing object
//
// • Register a "constructor function" that returns one pointer (or interface)
//
// Register an object
//
// Injecting an object means it has no dependencies, and will be used as a
// **shared** singleton instance for all resolutions within the graph.
//
//   type Fake struct {
//       Name string
//   }
//
//   g := dig.New()
//   err := g.Register(&Fake{Name: "I am an injected thing"})
//   require.NoError(t, err)
//
//   var f1 *Fake
//   err = g.Resolve(&f1)
//   require.NoError(t, err)
//
//   // f1 is ready to use here...
//
// Register a constructor
//
// This is a more interesting and widely used scenario. Constructor is defined as a
// function that returns exactly one pointer (or interface) and takes 0-N number of
// arguments. Each one of the arguments is automatically registered as a
// **dependency** and must also be an interface or a pointer.
//
// The following example illustrates injecting a constructor function for type
// *Object that requires *Dep to be present in the graph
//
//   g := dig.New()
//
//   type Dep struct{}
//   type Object struct{
//     Dep
//   }
//
//   func NewObject(d *Dep) *Object {
//     return &Object{Dep: d}
//   }
//
//   err := g.Register(NewObject)
//
// Resolve
//
// Resolve retrieves objects from the graph by type.
//
// There are future plans to do named retrievals to support multiple
// objects of the same type in the graph.
//
//
//   g := dig.New()
//
//   var o *Object
//   err := g.Resolve(&o) // notice the pointer to a pointer as param type
//   if err == nil {
//       // o is ready to use
//   }
//
//   type Do interface{}
//   var d Do
//   err := g.Resolve(&d) // notice pointer to an interface
//   if err == nil {
//       // d is ready to use
//   }
//
//
package dig
