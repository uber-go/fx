// Copyright (c) 2016 Uber Technologies, Inc.
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

package metrics

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

func TestRuntimeReporter(t *testing.T) {
	reporter := NewRuntimeReporter(tally.NoopScope, time.Millisecond)
	defer closeReporter(t, reporter)

	assert.False(t, reporter.IsRunning())
	reporter.Start()
	assert.True(t, reporter.IsRunning())
	runtime.GC()
	time.Sleep(5 * time.Millisecond)
	reporter.report()
}

func TestStartRuntimeReporter(t *testing.T) {
	reporter := StartReportingRuntimeMetrics(tally.NoopScope, time.Millisecond)
	defer closeReporter(t, reporter)

	assert.True(t, reporter.IsRunning())
	runtime.GC()
	time.Sleep(time.Millisecond)
	reporter.report()
	time.Sleep(time.Millisecond)
	reporter.report()
}

func TestStartRuntimeReporterStartAgain(t *testing.T) {
	reporter := StartReportingRuntimeMetrics(tally.NoopScope, time.Millisecond)
	defer closeReporter(t, reporter)

	assert.True(t, reporter.IsRunning())
	reporter.Start()
	assert.True(t, reporter.IsRunning())
}

func closeReporter(t *testing.T, r *RuntimeReporter) {
	r.Close()
	_, ok := <-r.quit
	assert.False(t, ok)
}
