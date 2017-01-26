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
	"fmt"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/internal/fxcontext"
	"go.uber.org/fx/service"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Filter applies filters on requests, request contexts or responses such as
// adding tracing to the context
type Filter interface {
	Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler)
}

// FilterFunc is an adaptor to call normal functions to apply filters
type FilterFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f FilterFunc) Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	f(ctx, w, r, next)
}

type tracingServerFilter struct {
	service.Host
}

func (f tracingServerFilter) Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	stopWatch := f.Host.Metrics().SubScope("http").SubScope("request").Timer("tracing").Start()
	operationName := r.Method
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		ctx.Logger().Info("Malformed inbound tracing context: ", "error", err.Error())
	}
	span := opentracing.GlobalTracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
	ext.HTTPUrl.Set(span, r.URL.String())
	defer func() {
		span.Finish()
		ctx.Logger().Debug("Span finished")
	}()

	ctx = ctx.(*fxcontext.Context).WithContextAwareLogger(span)
	ctx = &fxcontext.Context{
		Context: opentracing.ContextWithSpan(ctx, span),
	}
	r = r.WithContext(ctx)
	stopWatch.Stop()
	next.ServeHTTP(ctx, w, r)
}

// authorizationFilter authorizes services based on configuration
type authorizationFilter struct {
	service.Host
}

func (f authorizationFilter) Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	stopWatch := f.Host.Metrics().SubScope("http").SubScope("request").Timer("auth").Start()
	if err := f.Host.AuthClient().Authorize(ctx); err != nil {
		f.Host.Metrics().SubScope("http").SubScope("auth").Counter("fail").Inc(1)
		ctx.Logger().Error(auth.ErrAuthorization, "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Unauthorized access: %+v", err)
		stopWatch.Stop()
		return
	}
	stopWatch.Stop()
	next.ServeHTTP(ctx, w, r)
}

// panicFilter handles any panics and return an error
// panic filter should be added at the end of filter chain to catch panics
type panicFilter struct {
	service.Host
}

func (f panicFilter) Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Logger().Error("Panic recovered serving request", "error", err, "url", r.URL)
			f.Host.Metrics().SubScope("http").Counter("panic").Inc(1)
			w.Header().Add(ContentType, ContentTypeText)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Server error: %+v", err)
		}
	}()
	next.ServeHTTP(ctx, w, r)
}

// metricsFilter adds any default metrics related to HTTP
type metricsFilter struct {
	service.Host
}

func (f metricsFilter) Apply(ctx fx.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	stopwatch := f.Host.Metrics().SubScope("http").SubScope(r.Method).Timer("sla").Start()
	defer func() {
		f.Host.Metrics().SubScope("http").SubScope("status").Counter(w.Header().Get("Status")).Inc(1)
		stopwatch.Stop()
	}()
	next.ServeHTTP(ctx, w, r)
}
