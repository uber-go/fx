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

package rpc

import (
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/api/middleware"
)

const (
	_interceptorKey       = "yarpcUnaryInboundMiddleware"
	_onewayInterceptorKey = "yarpcOnewayInboundMiddleware"
)

// WithInboundMiddleware adds custom YARPC inboundMiddlewares to the module
func WithInboundMiddleware(i ...middleware.UnaryInbound) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddlewares := inboundMiddlewaresFromCreateInfo(*mci)
		inboundMiddlewares = append(inboundMiddlewares, i...)
		mci.Items[_interceptorKey] = inboundMiddlewares

		return nil
	}
}

// WithOnewayInboundMiddleware adds custom YARPC inboundMiddlewares to the module
func WithOnewayInboundMiddleware(i ...middleware.OnewayInbound) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddlewares := onewayInboundMiddlewaresFromCreateInfo(*mci)
		inboundMiddlewares = append(inboundMiddlewares, i...)
		mci.Items[_onewayInterceptorKey] = inboundMiddlewares
		return nil
	}
}

func inboundMiddlewaresFromCreateInfo(mci service.ModuleCreateInfo) []middleware.UnaryInbound {
	items, ok := mci.Items[_interceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-middleware slice to the data
	return items.([]middleware.UnaryInbound)
}

func onewayInboundMiddlewaresFromCreateInfo(mci service.ModuleCreateInfo) []middleware.OnewayInbound {
	items, ok := mci.Items[_onewayInterceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-middleware slice to the data
	return items.([]middleware.OnewayInbound)
}
