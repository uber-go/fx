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

package uhttp

import (
	"net/http"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/ulog"
)

type middlewareChain struct {
	currentMiddleware int
	finalHandler      http.Handler
	middlewares       []Middleware
}

func (fc middlewareChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fc.currentMiddleware == len(fc.middlewares) {
		fc.finalHandler.ServeHTTP(w, r)
	} else {
		middleware := fc.middlewares[fc.currentMiddleware]
		fc.currentMiddleware++
		middleware.Handle(w, r, fc)
	}
}

type middlewareChainBuilder struct {
	finalHandler http.Handler
	middlewares  []Middleware
}

func defaultMiddlewareChainBuilder(log ulog.Log, authClient auth.Client) middlewareChainBuilder {
	mcb := newMiddlewareChainBuilder()
	return mcb.AddMiddlewares(
		contextMiddleware{log},
		panicMiddleware{},
		metricsMiddleware{},
		tracingServerMiddleware{},
		authorizationMiddleware{
			authClient: authClient,
		})
}

// newMiddlewareChainBuilder creates an empty middlewareChainBuilder for setup
func newMiddlewareChainBuilder() middlewareChainBuilder {
	return middlewareChainBuilder{}
}

func (m middlewareChainBuilder) AddMiddlewares(middlewares ...Middleware) middlewareChainBuilder {
	for _, middleware := range middlewares {
		m.middlewares = append(m.middlewares, middleware)
	}
	return m
}

func (m middlewareChainBuilder) Build(finalHandler http.Handler) middlewareChain {
	return middlewareChain{
		middlewares:  m.middlewares,
		finalHandler: finalHandler,
	}
}
