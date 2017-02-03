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

// Package task is the Asynchronous Task Execution Module.
//
// The async task module presents a distributed task execution framework
// for services to execute a function asynchronously and durably.
//
//
// Backend
//
// Backends are messaging transports used by the framework to guarantee durability.
//
// Usage
//
// To use the module, initialize it at service startup and register any functions
// that will be invoked asynchronously. Call task.Enqueue on a function and the
// execution framework will send it to the backend implementation. Workers are
// running in parallel and listening to the backend. Once they receive a message
// from the backend, they will execute the function.
//
//
//   package main
//
//   import (
//   	"go.uber.org/fx/modules/task"
//     "go.uber.org/fx/service"
//   )
//
//   func main() {
//     svc, err := service.WithModules(
//       task.NewModule(newBackend),
//     ).Build()
//     svc.Start()
//     if err := task.Register(updateCache); err != nil {
//       // handle error
//     }
//   }
//
//   func newBackend(host service.Host) (task.Backend, error) {
//     b := // create backend here
//     return b, nil
//   }
//
//   func runActivity(input string) error {
//     // do things
//     results := "results"
//     if err := task.Enqueue(updateCache, input, results); err != nil {
//       return err
//     }
//   }
//
//   func updateCache(input string, results string) error {
//     // update cache with the name
//     return nil
//   }
//
// The async task module is a singleton and a service can intialize only one at this time.
// Users are free to define their own backends and encodings for message passing.
//
//
// Async function requirements
//
// For the function to be invoked asynchronously, the following criteria must be met:
// * The function should return only one value, which should be an error. The caller does not receive a
// return value from the called function.
// * The function should not take variadic arguments as input (support for this is coming soon).
//
//
//
package task
