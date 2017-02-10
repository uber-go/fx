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
	"go.uber.org/fx/modules/uhttp/stats"
	"go.uber.org/fx/service"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Filter applies filters on requests, request contexts or responses such as
// adding tracing to the context
type Filter interface {
	Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler)
}

// FilterFunc is an adaptor to call normal functions to apply filters
type FilterFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f FilterFunc) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	f(ctx, w, r, next)
}

type contextFilter struct {
	host service.Host
}

func (f contextFilter) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	ctx = fx.NewContext(ctx, f.host)
	next.ServeHTTP(ctx, w, r)
}

type tracingServerFilter struct {
}

func (f tracingServerFilter) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	operationName := r.Method
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		fx.Logger(ctx).Warn("Malformed inbound tracing context: ", "error", err.Error())
	}
	span := opentracing.GlobalTracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
	ext.HTTPUrl.Set(span, r.URL.String())
	defer span.Finish()

	ctx = opentracing.ContextWithSpan(ctx, span)

	ctx = fx.WithContextAwareLogger(ctx, span)

	r = r.WithContext(ctx)
	next.ServeHTTP(ctx, w, r)
}

// authorizationFilter authorizes services based on configuration
type authorizationFilter struct {
	authClient auth.Client
}

func (f authorizationFilter) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	if err := f.authClient.Authorize(ctx); err != nil {
		stats.HTTPAuthFailCounter.Inc(1)
		fx.Logger(ctx).Error(auth.ErrAuthorization, "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Unauthorized access: %+v", err)
		return
	}
	next.ServeHTTP(ctx, w, r)
}

// panicFilter handles any panics and return an error
// panic filter should be added at the end of filter chain to catch panics
type panicFilter struct{}

func (f panicFilter) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	defer func() {
		if err := recover(); err != nil {
			fx.Logger(ctx).Error("Panic recovered serving request", "error", err, "url", r.URL)
			stats.HTTPPanicCounter.Inc(1)
			w.Header().Add(ContentType, ContentTypeText)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Server error: %+v", err)
		}
	}()
	next.ServeHTTP(ctx, w, r)
}

// metricsFilter adds any default metrics related to HTTP
type metricsFilter struct {
}

func (f metricsFilter) Apply(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	defer stats.HTTPStatusCountScope.Tagged(map[string]string{stats.TagStatus: w.Header().Get("Status")}).Counter("total").Inc(1)
	next.ServeHTTP(ctx, w, r)
}
