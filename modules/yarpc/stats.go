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

package yarpc

import "github.com/uber-go/tally"

const (
	//_tagModule is module tag for metrics
	_tagModule = "module"
	// _tagType is either request or response
	_tagType = "type"
	// _tagProcedure is the procedure name
	_tagProcedure = "procedure"
	//_tagMiddleware is middleware type
	_tagMiddleware = "middleware"
)

var (
	rpcTags = map[string]string{
		_tagModule: "rpc",
		_tagType:   "request",
	}
)

type statsClient struct {
	rpcAuthFailCounter tally.Counter
	rpcHandleTimer     tally.Scope
	rpcPanicCounter    tally.Counter
}

func newStatsClient(scope tally.Scope) *statsClient {
	rpcTagsScope := scope.Tagged(rpcTags)
	return &statsClient{
		rpcTagsScope.Tagged(map[string]string{_tagMiddleware: "auth"}).Counter("fail"),
		rpcTagsScope.Tagged(rpcTags),
		rpcTagsScope.Counter("panic"),
	}
}

// RPCAuthFailCounter counts auth failures
func (c *statsClient) RPCAuthFailCounter() tally.Counter {
	return c.rpcAuthFailCounter
}

// RPCHandleTimer is a turnaround time for rpc handler
func (c *statsClient) RPCHandleTimer() tally.Scope {
	return c.rpcHandleTimer
}

// RPCPanicCounter counts panics occurred for rpc handler
func (c *statsClient) RPCPanicCounter() tally.Counter {
	return c.rpcPanicCounter
}
