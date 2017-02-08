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

// Package main is the Overview.
//
// [thriftsync] is a thriftrw plugin to identify and generate handlers for the given input
//  *.thrift file. With the use of thriftsync plugin, a user who needs to build a service should be able
//  to auto generate the code and write service specific logic without worrying about underlying platform.
//
//
// Example
//
// Following examples show how thriftsync syncs handler code with the updated thrift file:
//
// **New handler generation**
//
//   service TestService {
//     string testFunction(1: string param)
//   }
//
//   package main
//
//   import (
//     "context"
//
//     "testservice/testservice/testserviceserver"
//
//     "go.uber.org/fx/service"
//     "go.uber.org/yarpc/api/transport"
//   )
//
//   type YARPCHandler struct {
//     // TODO: modify the TestService handler with your suitable structure
//   }
//
//   // NewYARPCThriftHandler for your service
//   func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
//     handler := &YARPCHandler{}
//     return testserviceserver.New(handler), nil
//   }
//
//   func (h *YARPCHandler) TestFunction(ctx context.Context, param *string) (string, error) {
//     // TODO: write your code here
//     panic("To be implemented")
//   }
//
// **New function added to thrift file**
//
//   service TestService {
//     string testFunction(1: string param)
//
//     string newtestFunction(1: string param)
//   }
//
//   package main
//
//   import (
//     "context"
//
//     "testservice/testservice/testserviceserver"
//
//     "go.uber.org/fx/service"
//     "go.uber.org/yarpc/api/transport"
//   )
//
//   type YARPCHandler struct {
//     // TODO: modify the TestService handler with your suitable structure
//   }
//
//   // NewYARPCThriftHandler for your service
//   func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
//     handler := &YARPCHandler{}
//     return testserviceserver.New(handler), nil
//   }
//
//   func (h *YARPCHandler) testFunction(ctx context.Context, param string) (string, error) {
//     panic("To be implemented")
//   }
//
//   func (h *YARPCHandler) newtestFunction(ctx context.Context, param string) (string, error) {
//     panic("To be implemented")
//   }
//
// **New parameter added to a function**
//
//   service TestService {
//     string testFunction(1: string param)
//
//     string newtestFunction(1: string param, 2: string parameter2)
//   }
//
//   package main
//
//   import (
//     "context"
//
//     "testservice/testservice/testserviceserver"
//
//     "go.uber.org/fx/service"
//     "go.uber.org/yarpc/api/transport"
//   )
//
//   type YARPCHandler struct {
//     // TODO: modify the TestService handler with your suitable structure
//   }
//
//   // NewYARPCThriftHandler for your service
//   func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
//     handler := &YARPCHandler{}
//     return testserviceserver.New(handler), nil
//   }
//
//   func (h *YARPCHandler) testFunction(ctx context.Context, param string) (string, error) {
//     panic("To be implemented")
//   }
//
//   func (h *YARPCHandler) newtestFunction(ctx context.Context, param string, parameter2 string) (string, error) {
//     panic("To be implemented")
//   }
//
// **Updated parameter names and return types**
//
//   service TestService {
//     i64 testFunction(1: string newparameterName)
//
//     string newtestFunction(1: string param, 2: string parameter2)
//   }
//
//   package main
//
//   import (
//     "context"
//
//     "testservice/testservice/testserviceserver"
//
//     "go.uber.org/fx/service"
//     "go.uber.org/yarpc/api/transport"
//   )
//
//   type YARPCHandler struct {
//     // TODO: modify the TestService handler with your suitable structure
//   }
//
//   // NewYARPCThriftHandler for your service
//   func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
//     handler := &YARPCHandler{}
//     return testserviceserver.New(handler), nil
//   }
//
//   func (h *YARPCHandler) testFunction(ctx context.Context, newparameterName string) (int64, error) {
//     panic("To be implemented")
//   }
//
//   func (h *YARPCHandler) newtestFunction(ctx context.Context, param string, parameter2 string) (string, error) {
//     panic("To be implemented")
//   }
//
//
package main
