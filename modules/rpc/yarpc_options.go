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
	"go.uber.org/yarpc/transport"
)

const (
	_unaryInterceptorKey  = "yarpcUnaryInboundMiddleware"
	_onewayInterceptorKey = "yarpcOnewayInboundMiddleware"
)

// WithUnaryInboundMiddleware adds custom YARPC inboundMiddlewares to the module
func WithUnaryInboundMiddleware(i ...transport.UnaryInboundMiddleware) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddlewares := unaryInboundMiddlewaresFromCreateInfo(*mci)
		inboundMiddlewares = append(inboundMiddlewares, i...)
		mci.Items[_unaryInterceptorKey] = inboundMiddlewares

		return nil
	}
}

// WithOnewayInboundMiddleware adds custom YARPC inboundMid dlewares to the module
func WithOnewayInboundMiddleware(i ...transport.OnewayInboundMiddleware) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddlewares := onewayInboundMiddlewaresFromCreateInfo(*mci)
		inboundMiddlewares = append(inboundMiddlewares, i...)
		mci.Items[_onewayInterceptorKey] = inboundMiddlewares
		return nil
	}
}

func unaryInboundMiddlewaresFromCreateInfo(mci service.ModuleCreateInfo) []transport.UnaryInboundMiddleware {
	items, ok := mci.Items[_unaryInterceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-interceptor slice to the data
	return items.([]transport.UnaryInboundMiddleware)
}

func onewayInboundMiddlewaresFromCreateInfo(mci service.ModuleCreateInfo) []transport.OnewayInboundMiddleware {
	items, ok := mci.Items[_onewayInterceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-interceptor slice to the data
	return items.([]transport.OnewayInboundMiddleware)
}
