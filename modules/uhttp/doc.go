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

// Package uhttp is the HTTP Module.
//
// The HTTP module is built on top of Gorilla Mux (https://github.com/gorilla/mux),
// but the details of that are abstracted away through
// uhttp.RouteHandler.
//
//   package main
//
//   import (
//     "io"
//     "net/http"
//
//     "go.uber.org/fx"
//     "go.uber.org/fx/modules/uhttp"
//     "go.uber.org/fx/service"
//   )
//
//   func main() {
//     svc, err := service.WithModules(
//       uhttp.New(registerHTTP),
//     ).Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     svc.Start(true)
//   }
//
//   func registerHTTP(service service.Host) []uhttp.RouteHandler {
//     handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       fx.Logger(r.Context()).Info("Inside the handler")
//       io.WriteString(w, "Hello, world")
//     })
//
//     return []uhttp.RouteHandler{
//       uhttp.NewRouteHandler("/", handleHome)
//     }
//   }
//
// HTTP handlers are set up with filters that inject tracing, authentication information etc. into the
// request context. Request tracing, authentication and context-aware logging are set up by default.
// With context-aware logging, all log statements include trace information such as traceID and spanID.
// This allows service owners to easily find logs corresponding to a request within and even across services.
//
//
// HTTP Client
//
// The http client serves similar purpose as http module, but for making requests.
// It has a set of auth and tracing filters for http requests and a default timeout set to 2 minutes.
//
//
//   package main
//
//   import (
//     "log"
//     "net/http"
//
//     "go.uber.org/fx/modules/uhttp/client"
//     "go.uber.org/fx/service"
//   )
//
//   func main() {
//     svc, err := service.WithModules().Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     client := client.New(svc)
//     client.Get("https://www.uber.com")
//   }
//
// Benchmark results:
//
//   Current performance benchmark data:
//   BenchmarkClientFilters/empty-8         	100000000	        10.8 ns/op	       0 B/op	       0 allocs/op
//   BenchmarkClientFilters/tracing-8       	  500000	      3918 ns/op	    1729 B/op	      27 allocs/op
//   BenchmarkClientFilters/auth-8          	 1000000	      1866 ns/op	     719 B/op	      14 allocs/op
//   BenchmarkClientFilters/default-8       	  300000	      5604 ns/op	    2477 B/op	      41 allocs/op
//
//
package uhttp
