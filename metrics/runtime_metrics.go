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
	"sync/atomic"
	"time"

	"github.com/uber-go/tally"
)

// StartCollectingRuntimeMetrics starts generating runtime metrics under given metrics scope with
// given frequency.
// Recommended usage:
//     rmr := StartCollectingRuntimeMetrics(rootScope.Scope("runtime"), time.Second)
//     ...
//     rmr.Close()
func StartCollectingRuntimeMetrics(
	scope tally.Scope, collectInterval time.Duration, config RuntimeConfig,
) *RuntimeCollector {
	if config.Disabled {
		return nil
	}
	runtimeCollector := NewRuntimeCollector(scope, collectInterval)
	runtimeCollector.Start()
	return runtimeCollector
}

type runtimeMetrics struct {
	numGoRoutines   tally.Gauge
	goMaxProcs      tally.Gauge
	memoryAllocated tally.Gauge
	memoryHeap      tally.Gauge
	memoryHeapIdle  tally.Gauge
	memoryHeapInuse tally.Gauge
	memoryStack     tally.Gauge
	numGC           tally.Counter
	gcPauseMs       tally.Timer
	lastNumGC       uint32
}

// RuntimeCollector is a struct containing the state of the runtimeMetrics.
type RuntimeCollector struct {
	scope           tally.Scope
	collectInterval time.Duration
	metrics         runtimeMetrics
	started         bool
	quit            chan struct{}
}

// RuntimeConfig contains configuration for initializing runtime metrics
type RuntimeConfig struct {
	Disabled bool `yaml:"enabled"`
}

// NewRuntimeCollector creates a new RuntimeCollector.
func NewRuntimeCollector(
	scope tally.Scope, collectInterval time.Duration,
) *RuntimeCollector {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	return &RuntimeCollector{
		scope:           scope,
		collectInterval: collectInterval,
		metrics: runtimeMetrics{
			numGoRoutines:   scope.Gauge("num-goroutines"),
			goMaxProcs:      scope.Gauge("gomaxprocs"),
			memoryAllocated: scope.Gauge("memory.allocated"),
			memoryHeap:      scope.Gauge("memory.heap"),
			memoryHeapIdle:  scope.Gauge("memory.heapidle"),
			memoryHeapInuse: scope.Gauge("memory.heapinuse"),
			memoryStack:     scope.Gauge("memory.stack"),
			numGC:           scope.Counter("memory.num-gc"),
			gcPauseMs:       scope.Timer("memory.gc-pause-ms"),
			lastNumGC:       memstats.NumGC,
		},
		started: false,
		quit:    make(chan struct{}),
	}
}

// IsRunning returns true if the collector has been started and false if not.
func (r *RuntimeCollector) IsRunning() bool {
	return r.started
}

// Start starts the collector thread that periodically emits metrics.
func (r *RuntimeCollector) Start() {
	if r.started {
		return
	}
	go func() {
		ticker := time.NewTicker(r.collectInterval)
		for {
			select {
			case <-ticker.C:
				r.generate()
			case <-r.quit:
				ticker.Stop()
				return
			}
		}
	}()
	r.started = true
}

// generate sends runtime metrics to the local metrics collector.
func (r *RuntimeCollector) generate() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	r.metrics.numGoRoutines.Update(int64(runtime.NumGoroutine()))
	r.metrics.goMaxProcs.Update(int64(runtime.GOMAXPROCS(0)))
	r.metrics.memoryAllocated.Update(int64(memStats.Alloc))
	r.metrics.memoryHeap.Update(int64(memStats.HeapAlloc))
	r.metrics.memoryHeapIdle.Update(int64(memStats.HeapIdle))
	r.metrics.memoryHeapInuse.Update(int64(memStats.HeapInuse))
	r.metrics.memoryStack.Update(int64(memStats.StackInuse))

	// memStats.NumGC is a perpetually incrementing counter (unless it wraps at 2^32)
	num := memStats.NumGC
	lastNum := atomic.SwapUint32(&r.metrics.lastNumGC, num) // reset for the next iteration
	if delta := num - lastNum; delta > 0 {
		r.metrics.numGC.Inc(int64(delta))
		if delta > 255 {
			// too many GCs happened, the timestamps buffer got wrapped around. Generate only the last 256
			lastNum = num - 256
		}
		for i := lastNum; i != num; i++ {
			pause := memStats.PauseNs[i%256]
			r.metrics.gcPauseMs.Record(time.Duration(pause))
		}
	}
}

// Close stops collecting runtime metrics. It cannot be started again after it's
// been stopped.
func (r *RuntimeCollector) Close() {
	close(r.quit)
}
