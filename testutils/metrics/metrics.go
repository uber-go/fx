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
