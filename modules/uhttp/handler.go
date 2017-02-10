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
	"context"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/modules/uhttp/stats"
	"go.uber.org/fx/service"
)

// Handler is a context-aware extension of http.Handler.
type Handler interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request)

// ServeHTTP calls the caller HandlerFunc.
func (f HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

// Wrap the handler and host provided and return http.Handler for gorilla mux
func Wrap(host service.Host, handler Handler) http.Handler {
	return &handlerWrapper{
		host:    host,
		handler: handler,
	}
}

type handlerWrapper struct {
	host    service.Host
	handler Handler
}

// ServeHTTP calls Handler.ServeHTTP(ctx, w, r) and injects a new service context for use.
func (h *handlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := fx.NewContext(context.Background(), h.host)
	stopwatch := stats.HTTPMethodTimer[r.Method].Start()
	defer stopwatch.Stop()
	h.handler.ServeHTTP(ctx, w, r)
}
