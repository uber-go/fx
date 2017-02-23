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
	// TagModule is module tag for metrics
	TagModule = "module"
	// TagType is either request or response
	TagType = "type"
)

var (
	taskTags = map[string]string{
		TagModule: "task",
	}
)

// Client is a client for stats.
type Client struct {
	taskExecutionCount tally.Counter
	taskPublishCount   tally.Counter
	taskExecuteFail    tally.Counter
	taskPublishFail    tally.Counter
	taskExecutionTime  tally.Timer
	taskPublishTime    tally.Timer
}

// NewClient returns a new Client for the given tally.Scope.
func NewClient(scope tally.Scope) *Client {
	taskTagsScope := scope.Tagged(taskTags)
	return &Client{
		taskTagsScope.Tagged(map[string]string{TagType: "execution"}).Counter("count"),
		taskTagsScope.Tagged(map[string]string{TagType: "publish"}).Counter("count"),
		taskTagsScope.Tagged(map[string]string{TagType: "execution"}).Counter("fail"),
		taskTagsScope.Tagged(map[string]string{TagType: "publish"}).Counter("fail"),
		taskTagsScope.Tagged(map[string]string{TagType: "execution"}).Timer("time"),
		taskTagsScope.Tagged(map[string]string{TagType: "publish"}).Timer("time"),
	}
}

// TaskExecutionCount counts number of executions
func (c *Client) TaskExecutionCount() tally.Counter {
	return c.taskExecutionCount
}

// TaskPublishCount counts number of tasks enqueued
func (c *Client) TaskPublishCount() tally.Counter {
	return c.taskPublishCount
}

// TaskExecuteFail counts number of tasks failed to execute
func (c *Client) TaskExecuteFail() tally.Counter {
	return c.taskExecuteFail
}

// TaskPublishFail counts number of tasks failed to enqueue
func (c *Client) TaskPublishFail() tally.Counter {
	return c.taskPublishFail
}

// TaskExecutionTime is a turnaround time for execution
func (c *Client) TaskExecutionTime() tally.Timer {
	return c.taskExecutionTime
}

// TaskPublishTime is a publish time for tasks
func (c *Client) TaskPublishTime() tally.Timer {
	return c.taskPublishTime
}
