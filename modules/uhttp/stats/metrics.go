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

package stats

import "github.com/uber-go/tally"

const (
	//TagModule is module tag for metrics
	TagModule = "module"
	// TagType is either request or response
	TagType = "type"
	// TagStatus is status of the request
	TagStatus = "status"
	// TagMiddleware is the middleware tag
	TagMiddleware = "middleware"

	// MethodGet Get method
	MethodGet = "GET"
	// MethodHead Head method
	MethodHead = "HEAD"
	// MethodPost Post method
	MethodPost = "POST"
	// MethodPut Put method
	MethodPut = "PUT"
	// MethodPatch Patch method
	MethodPatch = "PATCH" // RFC 5789
	// MethodDelete Delete method
	MethodDelete = "DELETE"
	// MethodConnect Connect method
	MethodConnect = "CONNECT"
	// MethodOptions Options method
	MethodOptions = "OPTIONS"
	// MethodTrace Trace method
	MethodTrace = "TRACE"
)

// HTTPTags creates metrics scope with defined tags
var (
	HTTPTags = map[string]string{
		TagModule: "http",
		TagType:   "request",
	}

	// HTTPPanicCounter counts panics occurred in http
	HTTPPanicCounter tally.Counter
	// HTTPAuthFailCounter counts auth failures
	HTTPAuthFailCounter tally.Counter
	// HTTPMethodTimer is a turnaround time for http methods
	HTTPMethodTimer map[string]tally.Timer
	// HTTPStatusCountScope is a scope for http status
	HTTPStatusCountScope tally.Scope
)

// SetupHTTPMetrics allocates counters for necessary setup
func SetupHTTPMetrics(scope tally.Scope) {
	httpScope := scope.Tagged(HTTPTags)
	HTTPPanicCounter = httpScope.Counter("panic")

	HTTPAuthFailCounter = httpScope.Tagged(map[string]string{TagMiddleware: "auth"}).Counter("http.auth.fail")

	HTTPMethodTimer = make(map[string]tally.Timer)

	HTTPMethodTimer[MethodGet] = httpScope.Timer("http." + MethodGet + ".time")
	HTTPMethodTimer[MethodPost] = httpScope.Timer("http." + MethodPost + ".time")
	HTTPMethodTimer[MethodPut] = httpScope.Timer("http." + MethodPut + ".time")
	HTTPMethodTimer[MethodPatch] = httpScope.Timer("http." + MethodPatch + ".time")
	HTTPMethodTimer[MethodDelete] = httpScope.Timer("http." + MethodDelete + ".time")
	HTTPMethodTimer[MethodConnect] = httpScope.Timer("http." + MethodConnect + ".time")
	HTTPMethodTimer[MethodOptions] = httpScope.Timer("http." + MethodOptions + ".time")
	HTTPMethodTimer[MethodTrace] = httpScope.Timer("http." + MethodTrace + ".time")

	HTTPStatusCountScope = httpScope
}
