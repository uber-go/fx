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

// Package yarpc is the YARPC Module.
//
// The RPC module wraps YARPC (https://github.com/yarpc/yarpc-go) and exposes
// creators for both JSON- and Thrift-encoded messages.
//
//
// This module works in a way that's pretty similar to existing RPC projects:
//
// • Create an IDL file and run the appropriate tools on it (e.g. **thriftrw**) to
// generate the service and handler interfaces
//
//
// • Implement the service interface handlers as method receivers on a struct
//
// • Implement a top-level function, conforming to the
// yarpc.CreateThriftServiceFunc signature (fx/modules/yarpc/thrift.go that
// returns a
// []transport.Registrant YARPC implementation from the handler:
//
//   func NewMyServiceHandler(svc service.Host) ([]transport.Registrant, error) {
//     return myservice.New(&MyServiceHandler{}), nil
//   }
//
// • Pass that method into the module initialization:
//
//   func main() {
//     svc, err := service.WithModule(
//       yarpc.New(yarpc.CreateThriftServiceFunc(NewMyServiceHandler)),
//       service.WithModuleRole("service"),
//     ).Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     svc.Start(true)
//   }
//
// This will spin up the service.
//
//
package yarpc
