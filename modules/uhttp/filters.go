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

	"github.com/opentracing/opentracing-go"

	"golang.org/x/net/context"
)

// Filter applies filters on requests, request contexts or responses such as
// adding tracing to the context
type Filter interface {
	Apply(w http.ResponseWriter, r *http.Request, next http.Handler)
}

// FilterFunc is an adaptor to call normal functions to apply filters
type FilterFunc func(w http.ResponseWriter, r *http.Request, next http.Handler)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f FilterFunc) Apply(w http.ResponseWriter, r *http.Request, next http.Handler) {
	f(w, r, next)
}

type tracerFilter struct {
	tracer opentracing.Tracer
}

// Middlewares
func (t tracerFilter) Apply(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, tracingKey, t.tracer)
	r = r.WithContext(ctx)
	next.ServeHTTP(w, r)
}

// handle any panics and return an error
func errorFilter(w http.ResponseWriter, r *http.Request, next http.Handler) {
	defer func() {
		if err := recover(); err != nil {
			// TODO(ai) log and add stats to this
			w.Header().Add(ContentType, ContentTypeText)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Server error: %+v", err)
		}
	}()
	next.ServeHTTP(w, r)
}

type executionChain struct {
	currentFilter int
	filters       []Filter
	finalHandler  http.Handler
}

func (ec executionChain) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if ec.currentFilter < len(ec.filters) {
		filter := ec.filters[ec.currentFilter]
		ec.currentFilter++
		filter.Apply(w, req, ec)
	} else {
		ec.finalHandler.ServeHTTP(w, req)
	}
}
