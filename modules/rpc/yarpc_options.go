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

package rpc

import (
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/api/middleware"
)

const (
	_interceptorKey       = "yarpcUnaryInboundMiddleware"
	_onewayInterceptorKey = "yarpcOnewayInboundMiddleware"
)

// WithInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithInboundMiddleware(i ...middleware.UnaryInbound) service.ModuleOption {
	return service.WithModuleItem(_interceptorKey, func(existing interface{}) interface{} {
		var inboundMiddleware []middleware.UnaryInbound
		if existing != nil {
			inboundMiddleware = existing.([]middleware.UnaryInbound)
		}
		return append(inboundMiddleware, i...)
	})
}

// WithOnewayInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithOnewayInboundMiddleware(i ...middleware.OnewayInbound) service.ModuleOption {
	return service.WithModuleItem(_onewayInterceptorKey, func(existing interface{}) interface{} {
		var inboundMiddleware []middleware.OnewayInbound
		if existing != nil {
			inboundMiddleware = existing.([]middleware.OnewayInbound)
		}
		return append(inboundMiddleware, i...)
	})
}

func inboundMiddlewareFromModuleInfo(mci service.ModuleInfo) []middleware.UnaryInbound {
	if items, ok := mci.Item(_interceptorKey); ok {
		// Intentionally panic if programmer adds non-middleware slice to the data
		return items.([]middleware.UnaryInbound)
	}
	return nil
}

func onewayInboundMiddlewareFromModuleInfo(mci service.ModuleInfo) []middleware.OnewayInbound {
	if items, ok := mci.Item(_onewayInterceptorKey); ok {
		// Intentionally panic if programmer adds non-middleware slice to the data
		return items.([]middleware.OnewayInbound)
	}
	return nil
}
