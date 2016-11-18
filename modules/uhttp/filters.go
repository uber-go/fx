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
