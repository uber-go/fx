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

// HTTPTags creates metrics scope with defined tags
var (

	// RPCTags creates metrics scope with defined tags
	RPCTags = map[string]string{
		TagModule: "rpc",
		TagType:   "request",
	}

	// RPCAuthFailCounter counts auth failures
	RPCAuthFailCounter tally.Counter
	// RPCHandleTimer is a turnaround time for rpc handler
	RPCHandleTimer tally.Scope
)

// SetupRPCMetrics allocates counters for necessary setup
func SetupRPCMetrics(scope tally.Scope) {
	rpcTagsScope := scope.Tagged(RPCTags)
	RPCAuthFailCounter = rpcTagsScope.Tagged(map[string]string{TagMiddleware: "auth"}).Counter("rpc.auth.fail")
	RPCHandleTimer = rpcTagsScope.Tagged(RPCTags)
}
