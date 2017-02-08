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

import (
	"go.uber.org/fx/service"

	"github.com/uber-go/tally"
)

const (
	//TagModule is module tag for metrics
	TagModule = "module"
	// TagType is either request or response
	TagType = "type"
	// TagStatus is status of the request
	TagStatus = "status"
)

var httpMethods = []string{"GET", "HEAD", "PUT", "POST", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

// HTTPTags creates metrics scope with defined tags
var (
	HTTPTags = map[string]string{
		TagModule: "http",
		TagType:   "request",
	}

	// RPCTags creates metrics scope with defined tags
	RPCTags = map[string]string{
		TagModule: "rpc",
		TagType:   "request",
	}

	// TaskTags creates metrics scope with defined tags
	TaskTags = map[string]string{
		TagModule: "task",
		TagType:   "request",
	}
)

var (
	// HTTPPanicCounter counts panics occurred in http
	HTTPPanicCounter tally.Counter
	// HTTPAuthFailCounter counts auth failures
	HTTPAuthFailCounter tally.Counter
	// HTTPMethodTimer is a turnaround time for http methods
	HTTPMethodTimer map[string]tally.Timer
	// HTTPStatusCountScope is a scope for http status
	HTTPStatusCountScope tally.Scope

	// RPCAuthFailCounter counts auth failures
	RPCAuthFailCounter tally.Counter
	// RPCHandleTimer is a turnaround time for rpc handler
	RPCHandleTimer tally.Scope

	// TaskExecutionCount counts number of executions
	TaskExecutionCount tally.Counter
	// TaskPublishCount counts number of tasks enqueued
	TaskPublishCount tally.Counter
	// TaskExecuteFail counts number of tasks failed to execute
	TaskExecuteFail tally.Counter
	// TaskPublishFail counts number of tasks failed to enqueue
	TaskPublishFail tally.Counter
	// TaskExecutionTime is a turnaround time for execution
	TaskExecutionTime tally.Timer
	// TaskPublishTime is a publish time for tasks
	TaskPublishTime tally.Timer
)

// SetupHTTPMetrics allocates counters for necessary setup
func SetupHTTPMetrics(host service.Host) {
	HTTPPanicCounter = host.Metrics().Tagged(HTTPTags).Counter("panic")
	HTTPAuthFailCounter = host.Metrics().Tagged(HTTPTags).Counter("http.auth.fail")
	HTTPMethodTimer = make(map[string]tally.Timer)
	for _, method := range httpMethods {
		HTTPMethodTimer[method] = host.Metrics().Tagged(HTTPTags).Timer("http." + method + ".time")
	}
	HTTPStatusCountScope = host.Metrics().Tagged(HTTPTags)
}

// SetupRPCMetrics allocates counters for necessary setup
func SetupRPCMetrics(host service.Host) {
	RPCAuthFailCounter = host.Metrics().Tagged(RPCTags).Counter("rpc.auth.fail")
	RPCHandleTimer = host.Metrics().Tagged(RPCTags)
}

// SetupTaskMetrics allocates counters for necessary setup
func SetupTaskMetrics(host service.Host) {
	TaskExecutionCount = host.Metrics().Tagged(TaskTags).Counter("task.execution.count")
	TaskPublishCount = host.Metrics().Tagged(TaskTags).Counter("task.publish.count")
	TaskExecuteFail = host.Metrics().Tagged(TaskTags).Counter("task.execute.fail")
	TaskPublishFail = host.Metrics().Tagged(TaskTags).Counter("task.publish.fail")

	TaskExecutionTime = host.Metrics().Tagged(TaskTags).Timer("task.execution.time")
	TaskPublishTime = host.Metrics().Tagged(TaskTags).Timer("task.publish.time")
}
