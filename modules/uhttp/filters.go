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
	"fmt"
	"net/http"

	"go.uber.org/fx/service"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Filter applies filters on requests, request contexts or responses such as
// adding tracing to the context
type Filter interface {
	Apply(ctx service.Context, w http.ResponseWriter, r *http.Request, next Handler)
}

// FilterFunc is an adaptor to call normal functions to apply filters
type FilterFunc func(ctx service.Context, w http.ResponseWriter, r *http.Request, next Handler)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f FilterFunc) Apply(ctx service.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	f(ctx, w, r, next)
}

type tracerFilter struct{}

func (t tracerFilter) Apply(ctx service.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	operationName := r.Method
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	spanCtx, err := ctx.Tracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		ctx.Logger().Info("Malformed inbound tracing context: %s", err.Error())
	}
	span := ctx.Tracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
	ext.HTTPUrl.Set(span, r.URL.String())
	defer span.Finish()

	gctx := service.NewContext(opentracing.ContextWithSpan(ctx, span), ctx)
	next.ServeHTTP(gctx, w, r)
}

// panicFilter handles any panics and return an error
func panicFilter(ctx service.Context, w http.ResponseWriter, r *http.Request, next Handler) {
	defer func() {
		if err := recover(); err != nil {
			// TODO(ai) log and add stats to this
			w.Header().Add(ContentType, ContentTypeText)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Server error: %+v", err)
		}
	}()
	next.ServeHTTP(ctx, w, r)
}

func newExecutionChain(filters []Filter, finalHandler Handler) executionChain {
	return executionChain{
		filters:      filters,
		finalHandler: finalHandler,
	}
}

type executionChain struct {
	currentFilter int
	filters       []Filter
	finalHandler  Handler
}

func (ec executionChain) ServeHTTP(ctx service.Context, w http.ResponseWriter, req *http.Request) {
	if ec.currentFilter < len(ec.filters) {
		filter := ec.filters[ec.currentFilter]
		ec.currentFilter++
		filter.Apply(ctx, w, req, ec)
	} else {
		ec.finalHandler.ServeHTTP(ctx, w, req)
	}
}
