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
type Client interface {
	// HTTPPanicCounter counts panics occurred in http
	HTTPPanicCounter() tally.Counter
	// HTTPAuthFailCounter counts auth failures
	HTTPAuthFailCounter() tally.Counter
	// HTTPMethodTimer is a turnaround time for http methods
	HTTPMethodTimer() tally.Scope
	// HTTPStatusCountScope is a scope for http status
	HTTPStatusCountScope() tally.Scope
}

// NewClient returns a new Client for the given tally.Scope.
func NewClient(scope tally.Scope) Client {
	httpScope := scope.Tagged(httpTags)
	return &client{
		httpScope.Counter("panic"),
		httpScope.Tagged(map[string]string{TagMiddleware: "auth"}).Counter("fail"),
		httpScope.Tagged(httpTags),
		httpScope,
	}
}

type client struct {
	httpPanicCounter     tally.Counter
	httpAuthFailCounter  tally.Counter
	httpMethodTimer      tally.Scope
	httpStatusCountScope tally.Scope
}

func (c *client) HTTPPanicCounter() tally.Counter {
	return c.httpPanicCounter
}

func (c *client) HTTPAuthFailCounter() tally.Counter {
	return c.httpAuthFailCounter
}

func (c *client) HTTPMethodTimer() tally.Scope {
	return c.httpMethodTimer
}

func (c *client) HTTPStatusCountScope() tally.Scope {
	return c.httpStatusCountScope
}
