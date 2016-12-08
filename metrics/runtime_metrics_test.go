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
	testScope := tally.NewTestScope("", nil)
	// Setting this to second so that the only metrics generation call in this test comes from the
	// explicit call to generate(). Note that this will not actually cause a 1 sec delay since we
	// close the collector when this test ends.
	collector := NewRuntimeCollector(testScope, time.Second)
	defer closeCollector(t, collector)

	assert.False(t, collector.IsRunning())
	collector.Start()
	assert.True(t, collector.IsRunning())
	runtime.GC()
	collector.generate()
	verifyMetrics(t, testScope, true)
}

func TestStartRuntimeCollector(t *testing.T) {
	t.Skip()
	testScope := tally.NewTestScope("", nil)
	collector := StartCollectingRuntimeMetrics(testScope, time.Millisecond, _emptyRuntimeConfig)
	defer closeCollector(t, collector)

	assert.True(t, collector.IsRunning())
	time.Sleep(time.Millisecond)
	// Generate gets called
	time.Sleep(time.Millisecond)
	// Generate gets called
	verifyMetrics(t, testScope, false)
}

func TestStartRuntimeCollectorStartAgain(t *testing.T) {
	t.Skip()
	collector := StartCollectingRuntimeMetrics(tally.NoopScope, time.Millisecond, _emptyRuntimeConfig)
	defer closeCollector(t, collector)

	assert.True(t, collector.IsRunning())
	collector.Start()
	assert.True(t, collector.IsRunning())
}

func TestStartRuntimecollectorDisabled(t *testing.T) {
	t.Skip()
	config := RuntimeConfig{Disabled: true}
	collector := StartCollectingRuntimeMetrics(tally.NoopScope, time.Millisecond, config)
	assert.Nil(t, collector)
}

func closeCollector(t *testing.T, r *RuntimeCollector) {
	r.Close()
	_, ok := <-r.quit
	assert.False(t, ok)
}

func verifyMetrics(t *testing.T, scope tally.TestScope, withGC bool) {
	snapshot := scope.Snapshot()
	// Check gauges
	gauges := snapshot.Gauges()
	assert.NotNil(t, gauges["num-goroutines"].Value())
	assert.NotNil(t, gauges["gomaxprocs"].Value())
	assert.NotNil(t, gauges["memory.allocated"].Value())
	assert.NotNil(t, gauges["memory.heap"].Value())
	assert.NotNil(t, gauges["memory.heapidle"].Value())
	assert.NotNil(t, gauges["memory.heapinuse"].Value())
	assert.NotNil(t, gauges["memory.stack"].Value())
	if withGC {
		// Check counters
		counters := snapshot.Counters()
		assert.NotZero(t, counters["memory.num-gc"].Value())
		// Check timers
		timers := snapshot.Timers()
		assert.NotEmpty(t, timers["memory.gc-pause-ms"].Values())
	}
}
