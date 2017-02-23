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

type inboundMiddlewareChain struct {
	currentMiddleware int
	finalHandler      http.Handler
	middleware        []InboundMiddleware
}

func (fc inboundMiddlewareChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fc.currentMiddleware == len(fc.middleware) {
		fc.finalHandler.ServeHTTP(w, r)
	} else {
		middleware := fc.middleware[fc.currentMiddleware]
		fc.currentMiddleware++
		middleware.Handle(w, r, fc)
	}
}

type inboundMiddlewareChainBuilder struct {
	finalHandler http.Handler
	middleware   []InboundMiddleware
}

func defaultInboundMiddlewareChainBuilder(log ulog.Log, authClient auth.Client, statsClient *statsClient) inboundMiddlewareChainBuilder {
	mcb := newInboundMiddlewareChainBuilder()
	return mcb.AddMiddleware(
		contextInbound{log},
		panicInbound{statsClient},
		metricsInbound{statsClient},
		tracingInbound{},
		authorizationInbound{authClient, statsClient},
	)
}

// newInboundMiddlewareChainBuilder creates an empty middlewareChainBuilder for setup
func newInboundMiddlewareChainBuilder() inboundMiddlewareChainBuilder {
	return inboundMiddlewareChainBuilder{}
}

func (m inboundMiddlewareChainBuilder) AddMiddleware(middleware ...InboundMiddleware) inboundMiddlewareChainBuilder {
	m.middleware = append(m.middleware, middleware...)
	return m
}

func (m inboundMiddlewareChainBuilder) Build(finalHandler http.Handler) inboundMiddlewareChain {
	return inboundMiddlewareChain{
		middleware:   m.middleware,
		finalHandler: finalHandler,
	}
}
