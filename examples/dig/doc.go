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

// Package main is the Dependency Injection Graph Example.
//
// This example illustrates how package go.uber.org/fx/dig can be used to inject
// a dependency.
//
//
// struct HelloHandler exposes a dependency on hello.Sayer but does not
// provide any guidance of when or where to get it. On the other side package sayer
// injects an implementation of
// hello.Sayer interface into the graph.
//
// When the application is initialized, dig is asked to resolve the dependencies
// of the
// HelloHandler and create an instance.
//
// Run the example
//
//   $ go build
//   $ ./dig
//   $ curl localhost:8080
//   Well hello there DIG. How are you?
//
//
package main
