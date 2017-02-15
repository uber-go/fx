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

	"go.uber.org/fx/modules/uhttp/internal/stats"

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

// WithHost adds host to http.Handler and return http.Handler for gorilla mux.
func WithHost(host service.Host, handler http.Handler) http.Handler {
	return &handlerWithHost{
		host:    host,
		handler: handler,
	}
}

type handlerWithHost struct {
	host    service.Host
	handler http.Handler
}

// ServeHTTP calls Handler.ServeHTTP( w, r) and injects a new service context for use.
func (h *handlerWithHost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := ulog.NewLogContext(r.Context())
	stopwatch := stats.HTTPMethodTimer.Timer(r.Method).Start()
	defer stopwatch.Stop()

	h.handler.ServeHTTP(w, r.WithContext(ctx))
}
