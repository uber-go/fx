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
)

var (
	httpTags = map[string]string{
		TagModule: "http",
		TagType:   "request",
	}
)

// Client is a client for stats.
type Client struct {
	httpPanicCounter     tally.Counter
	httpAuthFailCounter  tally.Counter
	httpMethodTimer      tally.Scope
	httpStatusCountScope tally.Scope
}

// NewClient returns a new Client for the given tally.Scope.
func NewClient(scope tally.Scope) *Client {
	httpScope := scope.Tagged(httpTags)
	return &Client{
		httpScope.Counter("panic"),
		httpScope.Tagged(map[string]string{TagMiddleware: "auth"}).Counter("fail"),
		httpScope.Tagged(httpTags),
		httpScope,
	}
}

// HTTPPanicCounter counts panics occurred in http
func (c *Client) HTTPPanicCounter() tally.Counter {
	return c.httpPanicCounter
}

// HTTPAuthFailCounter counts auth failures
func (c *Client) HTTPAuthFailCounter() tally.Counter {
	return c.httpAuthFailCounter
}

// HTTPMethodTimer is a turnaround time for http methods
func (c *Client) HTTPMethodTimer() tally.Scope {
	return c.httpMethodTimer
}

// HTTPStatusCountScope is a scope for http status
func (c *Client) HTTPStatusCountScope() tally.Scope {
	return c.httpStatusCountScope
}
