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
	gcontext "context"
	"net/http"
	"regexp"
	"sync"

	"go.uber.org/fx/service"
)

// A RequestMatcher is a matching predicate for an HTTP Request
type RequestMatcher func(r *http.Request) bool

// PathMatchesRegexp returns a matcher that evaluates the path against the given regexp
func PathMatchesRegexp(re *regexp.Regexp) RequestMatcher {
	return func(r *http.Request) bool {
		return re.MatchString(r.URL.Path)
	}
}

type route struct {
	matches RequestMatcher
	handler Handler
}

// Router is an HTTP handler that can route based on arbitrary RequestMatchers.
// It applies Filter (or filter chain) before invoking the actual routes.
// If the filter is nil, it uses tracerFilter
type Router struct {
	mux    sync.RWMutex
	routes []route
	filter Filter
	host   service.Host
}

// NewRouter creates a new empty router
func NewRouter(host service.Host) *Router {
	return &Router{
		host: host,
	}
}

// ServeHTTP creates a new service.Context, and runs the filter chain,
// eventually routing to the appropriate handler based on the incoming path.
func (h *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := service.NewContext(gcontext.Background(), h.host)
	filter := h.filter
	if filter == nil {
		filter = tracerFilter{}
	}
	filter.Apply(ctx, w, r, HandlerFunc(h.serveHTTP))
}

// serveHTTP routes to the appropriate handler based on the incoming path
func (h *Router) serveHTTP(ctx service.Context, w http.ResponseWriter, r *http.Request) {
	var handler Handler

	h.mux.RLock()
	for idx := range h.routes {
		if h.routes[idx].matches(r) {
			handler = h.routes[idx].handler
			break
		}
	}
	h.mux.RUnlock()

	if handler != nil {
		handler.ServeHTTP(ctx, w, r)
	} else {
		http.NotFound(w, r)
	}
}

// AddPatternRoute is a convenience function for routing to the given handler
// based on a path regex.
func (h *Router) AddPatternRoute(pattern string, handler Handler) {
	h.AddRoute(PathMatchesRegexp(regexp.MustCompile(pattern)), handler)
}

// AddRoute adds a new router
func (h *Router) AddRoute(matches RequestMatcher, handler Handler) {
	h.mux.Lock()
	defer h.mux.Unlock()

	h.routes = append(h.routes, route{matches: matches, handler: handler})
}
