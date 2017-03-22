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
	"regexp"
	"sync"

	"go.uber.org/fx/service"
)

// The HandlerFunc type is an adapter to allow the use of
// functions as http handlers.
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// A RequestMatcher is a matching predicate for an HTTP Request
type RequestMatcher func(r *http.Request) bool

// PathMatchesRegexp returns a matcher that evaluates the path against the given regex
func PathMatchesRegexp(re *regexp.Regexp) RequestMatcher {
	return func(r *http.Request) bool {
		return re.MatchString(r.URL.Path)
	}
}

type route struct {
	matches RequestMatcher
	handler http.Handler
}

// Router is wrapper around gorila mux
type Router struct {
	mu         sync.RWMutex
	routes     []route
	middleware InboundMiddleware
	host       service.Host
}

// NewRouter creates a new empty router
func NewRouter(host service.Host) *Router {
	return &Router{
		host: host,
	}
}

// NewRouterWithMiddleware creates a new empty router with a pre-configured middleware
func NewRouterWithMiddleware(middleware InboundMiddleware) *Router {
	return &Router{middleware: middleware}
}

// ServeHTTP runs the middleware chain and routes to the
// appropriate handler
func (h *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	middleware := h.middleware
	if middleware == nil {
		middleware = DefaultMiddleware
	}
	middleware.Handle(w, r, HandlerFunc(h.serveHTTP))
}

// serveHTTP routes to the appropriate handler based on the routing path
func (h *Router) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler

	h.mu.RLock()
	for i := range h.routes {
		if h.routes[i].matches(r) {
			handler = h.routes[i].handler
			break
		}
	}
	h.mu.RUnlock()

	if handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// AddPatternRoute adds routing based on the regex
func (h *Router) AddPatternRoute(pattern string, handler http.Handler) {
	h.AddRoute(PathMatchesRegexp(regexp.MustCompile(pattern)), handler)
}

// AddRoute adds a new router
func (h *Router) AddRoute(matches RequestMatcher, handler http.Handler) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.routes = append(h.routes, route{matches: matches, handler: handler})
}
