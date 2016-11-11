// Copyright (c) 2016 Uber Technologies, Inc.
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

// Package uhttp is the HTTP Module.
//
// The HTTP module is built on top ofGorilla Mux (https://github.com/gorilla/mux),
// but the details of that are abstracted away through
//  uhttp.RouteHandler.
//
//   package main
//
//   import (
//     "io"
//     "net/http"
//
//     "go.uber.org/fx/modules/uhttp"
//     "go.uber.org/fx/service"
//   )
//
//   func main() {
//     service, err := service.WithModules(
//       uhttp.New(registerHTTP),
//     ).Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     service.Start(true)
//   }
//
//   func registerHTTP(service service.Host) []uhttp.RouteHandler {
//     handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       io.WriteString(w, "Hello, world")
//     })
//
//     return []uhttp.RouteHandler{
//       uhttp.NewRouteHandler("/", handleHome)
//     }
//   }
//
//
package uhttp
