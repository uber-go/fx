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
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/service"
)

type filterChain struct {
	currentFilter int
	finalHandler  Handler
	filters       []Filter
}

func newFilterChain(filters []Filter, finalHandler Handler) filterChain {
	return filterChain{
		finalHandler: finalHandler,
		filters:      filters,
	}
}

func (fc filterChain) ServeHTTP(ctx fx.Context, w http.ResponseWriter, r *http.Request) {
	if fc.currentFilter == len(fc.filters) {
		fc.finalHandler.ServeHTTP(ctx, w, r)
	} else {
		filter := fc.filters[fc.currentFilter]
		fc.currentFilter++
		filter.Apply(ctx, w, r, fc)
	}
}

// FilterChainBuilder builds a filterChain object with added filters
type FilterChainBuilder interface {
	// AddFilter is used to add the next filter to the chain during construction time.
	// The calls to AddFilter can be chained.
	AddFilter(filter Filter) FilterChainBuilder

	// Build creates an immutable FilterChain.
	Build(finalHandler Handler) filterChain
}

type filterChainBuilder struct {
	service.Host

	finalHandler Handler
	filters      []Filter
}

func defaultFilterChainBuilder(host service.Host) FilterChainBuilder {
	fcb := NewFilterChainBuilder(host)
	return fcb.AddFilter(contextFilter(host)).
		AddFilter(tracingServerFilter(host)).
		AddFilter(authorizationFilter(host)).
		AddFilter(panicFilter(host))
}

// NewFilterChainBuilder creates an empty filterChainBuilder for setup
func NewFilterChainBuilder(host service.Host) FilterChainBuilder {
	return &filterChainBuilder{
		Host: host,
	}
}

func (f filterChainBuilder) AddFilter(filter Filter) FilterChainBuilder {
	f.filters = append(f.filters, filter)
	return f
}

func (f filterChainBuilder) Build(finalHandler Handler) filterChain {
	fc := filterChain{}
	for _, ff := range f.filters {
		fc.filters = append(fc.filters, ff)
	}
	fc.finalHandler = finalHandler
	return fc
}
