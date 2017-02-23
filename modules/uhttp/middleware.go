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
	"fmt"
	"net/http"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/modules/uhttp/internal/stats"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

const _panicResponse = "Server Error"

// InboundMiddleware applies inbound middleware on requests or responses such as
// adding tracing to the context.
type InboundMiddleware interface {
	Handle(w http.ResponseWriter, r *http.Request, next http.Handler)
}

// InboundMiddlewareFunc is an adaptor to call normal functions to apply inbound middleware.
type InboundMiddlewareFunc func(w http.ResponseWriter, r *http.Request, next http.Handler)

// Handle implements Handle from the InboundMiddleware interface and simply delegates to the function
func (f InboundMiddlewareFunc) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	f(w, r, next)
}

type contextInbound struct {
	log ulog.Log
}

func (f contextInbound) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx := ulog.ContextWithLogger(r.Context(), f.log)
	next.ServeHTTP(w, r.WithContext(ctx))
}

type tracingInbound struct{}

func (f tracingInbound) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx := r.Context()
	operationName := r.Method
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		ulog.Logger(ctx).Warn("Malformed inbound tracing context: ", "error", err.Error())
	}
	span := opentracing.GlobalTracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
	ext.HTTPUrl.Set(span, r.URL.String())
	defer span.Finish()

	ctx = opentracing.ContextWithSpan(ctx, span)

	ctx = ulog.ContextWithTraceLogger(ctx, span)

	next.ServeHTTP(w, r.WithContext(ctx))
}

// authorizationInbound authorizes services based on configuration
type authorizationInbound struct {
	authClient  auth.Client
	statsClient stats.Client
}

func (f authorizationInbound) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if err := f.authClient.Authorize(r.Context()); err != nil {
		f.statsClient.HTTPAuthFailCounter().Inc(1)
		ulog.Logger(r.Context()).Error(auth.ErrAuthorization, "error", err)
		http.Error(w, fmt.Sprintf("Unauthorized access: %+v", err), http.StatusUnauthorized)
		return
	}
	next.ServeHTTP(w, r)
}

// panicInbound handles any panics and return an error
// panic inbound middleware should be added at the end of middleware chain to catch panics
type panicInbound struct {
	statsClient stats.Client
}

func (f panicInbound) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx := r.Context()
	defer func() {
		if err := recover(); err != nil {
			ulog.Logger(ctx).Error("Panic recovered serving request", "error", errors.Errorf("panic in handler: %+v", err), "url", r.URL)
			f.statsClient.HTTPPanicCounter().Inc(1)
			http.Error(w, _panicResponse, http.StatusInternalServerError)
		}
	}()
	next.ServeHTTP(w, r)
}

// metricsInbound adds any default metrics related to HTTP
type metricsInbound struct {
	statsClient stats.Client
}

func (f metricsInbound) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	stopwatch := f.statsClient.HTTPMethodTimer().Timer(r.Method).Start()
	defer stopwatch.Stop()
	defer f.statsClient.HTTPStatusCountScope().Tagged(map[string]string{stats.TagStatus: w.Header().Get("Status")}).Counter("total").Inc(1)
	next.ServeHTTP(w, r)
}
