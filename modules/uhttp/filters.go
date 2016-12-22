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

package uhttp

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/internal/fxcontext"
	"go.uber.org/fx/service"
	"go.uber.org/fx/uauth"

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

// contextFilter updates context to fx.Context
func contextFilter(host service.Host) FilterFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
		fxctx := fxcontext.New(ctx, host)
		next.ServeHTTP(fxctx, w, r)
	}
}

func tracingServerFilter(host service.Host) FilterFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
		// TODO:(anup) GFM-257 benchmark performance comparing with using type assertion
		fxctx := &fxcontext.Context{
			Context: ctx,
		}
		operationName := r.Method
		carrier := opentracing.HTTPHeadersCarrier(r.Header)
		spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			fxctx.Logger().Info("Malformed inbound tracing context: ", "error", err.Error())
		}

		span := opentracing.GlobalTracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
		ext.HTTPUrl.Set(span, r.URL.String())
		defer span.Finish()

		fxctx = &fxcontext.Context{
			Context: opentracing.ContextWithSpan(ctx, span),
		}
		r = r.WithContext(fxctx)
		next.ServeHTTP(fxctx, w, r)
	}
}

// authorizationFilter authorizes services based on configuration
func authorizationFilter(host service.Host) FilterFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
		fxctx := &fxcontext.Context{
			Context: ctx,
		}
		authClient := uauth.Instance()
		err := authClient.Authorize(fxctx)
		if err != nil {
			host.Metrics().SubScope("http").SubScope("auth").Counter("fail").Inc(1)
			fxctx.Logger().Error(uauth.ErrAuthorization, "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Unauthorized access: %+v", err)
			return
		}
		next.ServeHTTP(fxctx, w, r)
	}
}

// panicFilter handles any panics and return an error
func panicFilter(host service.Host) FilterFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) {
		fxctx := &fxcontext.Context{
			Context: ctx,
		}
		defer func() {
			if err := recover(); err != nil {
				fxctx.Logger().Error("Panic recovered serving request", "error", err, "url", r.URL)
				host.Metrics().Counter("http.panic").Inc(1)
				w.Header().Add(ContentType, ContentTypeText)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Server error: %+v", err)
			}
		}()
		next.ServeHTTP(fxctx, w, r)
	}
}

func newFilterChain(filters []Filter, finalHandler Handler) filterChain {
	return filterChain{
		filters:      filters,
		finalHandler: finalHandler,
	}
}

type filterChain struct {
	currentFilter int
	filters       []Filter
	finalHandler  Handler
}

func (ec filterChain) ServeHTTP(ctx fx.Context, w http.ResponseWriter, req *http.Request) {
	if ec.currentFilter < len(ec.filters) {
		filter := ec.filters[ec.currentFilter]
		ec.currentFilter++
		filter.Apply(ctx, w, req, ec)
	} else {
		ec.finalHandler.ServeHTTP(ctx, w, req)
	}
}
