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
	"sync"
	"time"

	"github.com/uber-go/tally"
)

// TestStatsReporter allows for easy testing of any metrics related functionality
type TestStatsReporter struct {
	m sync.Mutex

	Cw sync.WaitGroup
	Gw sync.WaitGroup
	Tw sync.WaitGroup

	Counters map[string]int64
	Gauges   map[string]float64
	Timers   map[string]time.Duration
}

// ReportCounter collects all the counters into a map
func (r *TestStatsReporter) ReportCounter(name string, tags map[string]string, value int64) {
	r.m.Lock()
	r.Counters[name] += value
	r.Cw.Done()
	r.m.Unlock()
}

// ReportGauge collects all the gauges into a map
func (r *TestStatsReporter) ReportGauge(name string, tags map[string]string, value float64) {
	r.m.Lock()
	r.Gauges[name] = value
	r.Gw.Done()
	r.m.Unlock()
}


// ReportTimer collects all the timers into a map
func (r *TestStatsReporter) ReportTimer(name string, tags map[string]string, interval time.Duration) {
	r.m.Lock()
	r.Timers[name] = interval
	r.Tw.Done()
	r.m.Unlock()
}

// Capabilities is a nop in this case
func (r *TestStatsReporter) Capabilities() tally.Capabilities {
	return nil
}

// Flush is a nop
func (r *TestStatsReporter) Flush() {}

// NewTestStatsReporter returns new instance of test stats reporter
func NewTestStatsReporter() *TestStatsReporter {
	return &TestStatsReporter{
		Counters: make(map[string]int64),
		Gauges:   make(map[string]float64),
		Timers:   make(map[string]time.Duration),
	}
}

// NewTestScope returns a pair of scope and reporter that can be used for testing
func NewTestScope() (tally.Scope, *TestStatsReporter) {
	r := NewTestStatsReporter()
	scope, _ := tally.NewRootScope("", nil, r, 100*time.Millisecond)
	return scope, r
}
