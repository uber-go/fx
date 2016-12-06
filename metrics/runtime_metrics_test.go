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

var _emptyRuntimeConfig = RuntimeConfig{}

func TestRuntimeCollector(t *testing.T) {
	collector := NewRuntimeCollector(tally.NoopScope, time.Millisecond)
	defer closeCollector(t, collector)

	assert.False(t, collector.IsRunning())
	collector.Start()
	assert.True(t, collector.IsRunning())
	runtime.GC()
	time.Sleep(5 * time.Millisecond)
	collector.generate()
}

func TestStartRuntimeCollector(t *testing.T) {
	collector := StartCollectingRuntimeMetrics(tally.NoopScope, time.Millisecond, _emptyRuntimeConfig)
	defer closeCollector(t, collector)

	assert.True(t, collector.IsRunning())
	runtime.GC()
	time.Sleep(time.Millisecond)
	// Generate gets called
	time.Sleep(time.Millisecond)
	// Generate gets called
}

func TestStartRuntimeCollectorStartAgain(t *testing.T) {
	collector := StartCollectingRuntimeMetrics(tally.NoopScope, time.Millisecond, _emptyRuntimeConfig)
	defer closeCollector(t, collector)

	assert.True(t, collector.IsRunning())
	collector.Start()
	assert.True(t, collector.IsRunning())
}

func TestStartRuntimecollectorDisabled(t *testing.T) {
	config := RuntimeConfig{Disabled: true}
	collector := StartCollectingRuntimeMetrics(tally.NoopScope, time.Millisecond, config)
	assert.Nil(t, collector)
}

func closeCollector(t *testing.T, r *RuntimeCollector) {
	r.Close()
	_, ok := <-r.quit
	assert.False(t, ok)
}
