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
	// TagProcedure is the procedure name
	TagProcedure = "procedure"
	//TagMiddleware is middleware type
	TagMiddleware = "middleware"
)

var (
	rpcTags = map[string]string{
		TagModule: "rpc",
		TagType:   "request",
	}
)

// Client is a client for stats.
type Client interface {
	// RPCAuthFailCounter counts auth failures
	RPCAuthFailCounter() tally.Counter
	// RPCHandleTimer is a turnaround time for rpc handler
	RPCHandleTimer() tally.Scope
	// RPCPanicCounter counts panics occurred for rpc handler
	RPCPanicCounter() tally.Counter
}

func NewClient(scope tally.Scope) Client {
	rpcTagsScope := scope.Tagged(rpcTags)
	return &client{
		rpcTagsScope.Tagged(map[string]string{TagMiddleware: "auth"}).Counter("fail"),
		rpcTagsScope.Tagged(rpcTags),
		rpcTagsScope.Counter("panic"),
	}
}

type client struct {
	rpcAuthFailCounter tally.Counter
	rpcHandleTimer     tally.Scope
	rpcPanicCounter    tally.Counter
}

func (c *client) RPCAuthFailCounter() tally.Counter {
	return c.rpcAuthFailCounter
}

func (c *client) RPCHandleTimer() tally.Scope {
	return c.rpcHandleTimer
}

func (c *client) RPCPanicCounter() tally.Counter {
	return c.rpcPanicCounter
}
