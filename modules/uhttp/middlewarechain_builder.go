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
	"sync"
)

// DefaultMiddleware is used by Router if no custom middleware are provided.
var DefaultMiddleware = newInboundMiddlewareChainBuilder().
	AddMiddleware(InboundMiddlewareFunc(nopMiddleware)).
	Build()

func newInboundMiddlewareChainBuilder() *inboundMiddlewareChainBuilder {
	return &inboundMiddlewareChainBuilder{}
}

type inboundMiddlewareChainBuilder struct {
	middlewares []InboundMiddleware
}

func (b *inboundMiddlewareChainBuilder) AddMiddleware(middleware ...InboundMiddleware) *inboundMiddlewareChainBuilder {
	b.middlewares = append(b.middlewares, middleware...)
	return b
}

func (b *inboundMiddlewareChainBuilder) Build() InboundMiddleware {
	return &inboundMiddlewareChain{
		middlewares: b.middlewares,
		pool: sync.Pool{
			New: func() interface{} {
				return &chainExecution{}
			},
		},
	}
}

type inboundMiddlewareChain struct {
	middlewares []InboundMiddleware
	pool        sync.Pool
}

type chainExecution struct {
	middlewares       []InboundMiddleware
	finalHandler      http.Handler
	currentMiddleware int
}

func (chain *inboundMiddlewareChain) Handle(w http.ResponseWriter, r *http.Request, handler http.Handler) {
	if len(chain.middlewares) == 0 {
		handler.ServeHTTP(w, r)
		return
	}
	exec := chain.pool.Get().(*chainExecution)
	exec.middlewares = chain.middlewares
	exec.currentMiddleware = 0
	exec.finalHandler = handler
	exec.ServeHTTP(w, r)
	chain.pool.Put(exec)
}

func (exec *chainExecution) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if exec.currentMiddleware == len(exec.middlewares) {
		exec.finalHandler.ServeHTTP(w, r)
	} else {
		middleware := exec.middlewares[exec.currentMiddleware]
		exec.currentMiddleware++
		middleware.Handle(w, r, exec)
	}
}

func nopMiddleware(w http.ResponseWriter, r *http.Request, next http.Handler) {
	next.ServeHTTP(w, r)
}
