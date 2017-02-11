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

	"go.uber.org/fx/service"
)

type filterChain struct {
	currentFilter int
	finalHandler  http.Handler
	filters       []Filter
}

func (fc filterChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fc.currentFilter == len(fc.filters) {
		fc.finalHandler.ServeHTTP(w, r)
	} else {
		filter := fc.filters[fc.currentFilter]
		fc.currentFilter++
		filter.Apply(w, r, fc)
	}
}

type filterChainBuilder struct {
	service.Host

	finalHandler http.Handler
	filters      []Filter
}

func defaultFilterChainBuilder(host service.Host) filterChainBuilder {
	fcb := newFilterChainBuilder(host)
	return fcb.AddFilters(
		contextFilter{host},
		panicFilter{},
		metricsFilter{},
		tracingServerFilter{},
		authorizationFilter{
			authClient: host.AuthClient(),
		})
}

// NewFilterChainBuilder creates an empty filterChainBuilder for setup
func newFilterChainBuilder(host service.Host) filterChainBuilder {
	return filterChainBuilder{
		Host: host,
	}
}

func (f filterChainBuilder) AddFilters(filters ...Filter) filterChainBuilder {
	for _, filter := range filters {
		f.filters = append(f.filters, filter)
	}
	return f
}

func (f filterChainBuilder) Build(finalHandler http.Handler) filterChain {
	return filterChain{
		filters:      f.filters,
		finalHandler: finalHandler,
	}
}
