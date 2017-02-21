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
	"go.uber.org/fx/dig"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/api/middleware"
)

const (
	_interceptorKey       = "yarpcUnaryInboundMiddleware"
	_onewayInterceptorKey = "yarpcOnewayInboundMiddleware"
	_graphInterceptorKey  = "yarpcDIGraph"
)

// WithInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithInboundMiddleware(i ...middleware.UnaryInbound) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddleware := inboundMiddlewareFromCreateInfo(*mci)
		inboundMiddleware = append(inboundMiddleware, i...)
		mci.Items[_interceptorKey] = inboundMiddleware

		return nil
	}
}

// WithOnewayInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithOnewayInboundMiddleware(i ...middleware.OnewayInbound) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		inboundMiddleware := onewayInboundMiddlewareFromCreateInfo(*mci)
		inboundMiddleware = append(inboundMiddleware, i...)
		mci.Items[_onewayInterceptorKey] = inboundMiddleware
		return nil
	}
}

func inboundMiddlewareFromCreateInfo(mci service.ModuleCreateInfo) []middleware.UnaryInbound {
	items, ok := mci.Items[_interceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-middleware slice to the data
	return items.([]middleware.UnaryInbound)
}

func onewayInboundMiddlewareFromCreateInfo(mci service.ModuleCreateInfo) []middleware.OnewayInbound {
	items, ok := mci.Items[_onewayInterceptorKey]
	if !ok {
		return nil
	}

	// Intentionally panic if programmer adds non-middleware slice to the data
	return items.([]middleware.OnewayInbound)
}

func withGraph(graph dig.Graph) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		mci.Items[_graphInterceptorKey] = graph
		return nil
	}
}

func graphFromCreateInfo(mci service.ModuleCreateInfo) dig.Graph {
	g, ok := mci.Items[_graphInterceptorKey]
	if !ok {
		return dig.DefaultGraph()
	}

	// Intentionally panic if programmer adds non-middleware slice to the data
	return g.(dig.Graph)
}
