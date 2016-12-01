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

// StartReportingRuntimeMetrics starts reporting runtime metrics under given metrics scope with
// given frequency.
// This is just a convenience wrapper around creating and starting a new reporter manually.
// Recommended usage:
//     StartReportingRuntimeMetrics(rootScope.Scope("runtime"), time.Second)
// If you want to stop the reporter later:
//     rmr := StartReportingRuntimeMetrics(rootScope.Scope("runtime"), time.Second)
//     ...
//     rmr.Close()
func StartReportingRuntimeMetrics(
	scope tally.Scope, reportInterval time.Duration,
) *RuntimeReporter {
	runtimeReporter := NewRuntimeReporter(scope, reportInterval)
	runtimeReporter.Start()
	return runtimeReporter
}

type runtimeMetrics struct {
	NumGoRoutines   tally.Gauge
	GoMaxProcs      tally.Gauge
	MemoryAllocated tally.Gauge
	MemoryHeap      tally.Gauge
	MemoryHeapIdle  tally.Gauge
	MemoryHeapInuse tally.Gauge
	MemoryStack     tally.Gauge
	NumGC           tally.Counter
	GcPauseMs       tally.Timer
	lastNumGC       uint32
}

// RuntimeReporter is a struct containing the state of the RuntimeMetricsReporter.
type RuntimeReporter struct {
	scope          tally.Scope
	reportInterval time.Duration
	metrics        runtimeMetrics
	started        bool
	quit           chan struct{}
}

// NewRuntimeReporter creates a new RuntimeMetricsReporter.
func NewRuntimeReporter(
	scope tally.Scope, reportInterval time.Duration,
) *RuntimeReporter {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	return &RuntimeReporter{
		scope:          scope,
		reportInterval: reportInterval,
		metrics: runtimeMetrics{
			NumGoRoutines:   scope.Gauge("num-goroutines"),
			GoMaxProcs:      scope.Gauge("gomaxprocs"),
			MemoryAllocated: scope.Gauge("memory.allocated"),
			MemoryHeap:      scope.Gauge("memory.heap"),
			MemoryHeapIdle:  scope.Gauge("memory.heapidle"),
			MemoryHeapInuse: scope.Gauge("memory.heapinuse"),
			MemoryStack:     scope.Gauge("memory.stack"),
			NumGC:           scope.Counter("memory.num-gc"),
			GcPauseMs:       scope.Timer("memory.gc-pause-ms"),
			lastNumGC:       memstats.NumGC,
		},
		started: false,
		quit:    make(chan struct{}),
	}
}

// IsRunning returns true if the reporter has been started and false if not.
func (r *RuntimeReporter) IsRunning() bool {
	return r.started
}

// Start starts the reporter thread that periodically emits metrics.
func (r *RuntimeReporter) Start() {
	if r.started {
		return
	}
	go func() {
		ticker := time.NewTicker(r.reportInterval)
		for {
			select {
			case <-ticker.C:
				r.report()
			case <-r.quit:
				ticker.Stop()
				return
			}
		}
	}()
	r.started = true
}

// report Sends runtime metrics to the local metrics collector.
func (r *RuntimeReporter) report() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	r.metrics.NumGoRoutines.Update(int64(runtime.NumGoroutine()))
	r.metrics.GoMaxProcs.Update(int64(runtime.GOMAXPROCS(0)))
	r.metrics.MemoryAllocated.Update(int64(memStats.Alloc))
	r.metrics.MemoryHeap.Update(int64(memStats.HeapAlloc))
	r.metrics.MemoryHeapIdle.Update(int64(memStats.HeapIdle))
	r.metrics.MemoryHeapInuse.Update(int64(memStats.HeapInuse))
	r.metrics.MemoryStack.Update(int64(memStats.StackInuse))

	// memStats.NumGC is a perpetually incrementing counter (unless it wraps at 2^32)
	num := memStats.NumGC
	lastNum := atomic.SwapUint32(&r.metrics.lastNumGC, num) // reset for the next iteration
	if delta := num - lastNum; delta > 0 {
		r.metrics.NumGC.Inc(int64(delta))
		if delta > 255 {
			// too many GCs happened, the timestamps buffer got wrapped around. Report only the last 256
			lastNum = num - 256
		}
		for i := lastNum; i != num; i++ {
			pause := memStats.PauseNs[i%256]
			r.metrics.GcPauseMs.Record(time.Duration(pause))
		}
	}
}

// Close stops reporting runtime metrics. The reporter cannot be started again after it's
// been stopped.
func (r *RuntimeReporter) Close() {
	close(r.quit)
}
