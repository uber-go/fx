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

package main

import (
	"fmt"
	"io"
	"net/http"

	"go.uber.org/fx/modules/uhttp"
	"go.uber.org/fx/service"
)

type exampleHandler struct{}

func (exampleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, fmt.Sprintf("Headers: %+v", r.Header))
}

func enforceHeader(r uhttp.Route) uhttp.Route {
	// require some weird headers
	return r.Headers("X-Uber-FX", "yass")
}

func registerHTTPers(service service.Host) []uhttp.RouteHandler {
	handler := &exampleHandler{}
	return []uhttp.RouteHandler{
		uhttp.NewRouteHandler("/", handler),
	}
}

type simpleFilter struct{}

func (simpleFilter) Apply(w http.ResponseWriter, r *http.Request, next http.Handler) {
	io.WriteString(w, "Going through simpleFilter")
	next.ServeHTTP(w, r)
}
