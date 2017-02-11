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

	"github.com/gorilla/mux"
)

// FromGorilla turns a gorilla mux route into an UberFx route
func FromGorilla(r *mux.Route) Route {
	return Route{
		r: r,
	}
}

// A RouteHandler is an HTTP handler for a single route
type RouteHandler struct {
	Path    string
	Handler http.Handler
}

// NewRouteHandler creates a route handler
func NewRouteHandler(path string, handler http.Handler) RouteHandler {
	return RouteHandler{
		Path:    path,
		Handler: handler,
	}
}

// A Route represents a handler for HTTP requests, with restrictions
type Route struct {
	r *mux.Route
}

// GorillaMux returns the underlying mux if you need to use it directly
func (r Route) GorillaMux() *mux.Route {
	return r.r
}

// Headers allows easy enforcement of headers
func (r Route) Headers(headerPairs ...string) Route {
	return Route{
		r.r.Headers(headerPairs...),
	}
}

// Methods allows easy enforcement of metthods (HTTP Verbs)
func (r Route) Methods(methods ...string) Route {
	return Route{
		r.r.Methods(methods...),
	}
}
