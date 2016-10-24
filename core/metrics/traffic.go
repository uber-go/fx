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

// TODO(glib): Do we need this file?
// With a tally StatsReporter we shouldn't have to custom roll anything else
// It will be able to answer questions about traffic

package metrics

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx/core/ulog"
)

const (
	// TrafficCorrelationID is a key for handlers to correlate a request
	TrafficCorrelationID = "traffic_id"
)

// A TrafficReporter handles reporting traffic data for requests
type TrafficReporter interface {
	Start(name string, data map[string]string, timeout time.Duration) TrafficTracker
}

// A NoopTrafficReporter is used for tests and places where you don't care about reporting traffic
type NoopTrafficReporter struct{}

// Start is used for a NoopTrafficReporter to discard traffic results
func (NoopTrafficReporter) Start(name string, data map[string]string, timeout time.Duration) TrafficTracker {
	callback := func(name, desc string, elapsed time.Duration, data map[string]string, result interface{}, err error) {
		// Do Nothing
	}
	return NewDefaultTrafficTracker(name, nil, timeout, callback)
}

// A TrafficTracker is used for a single request to track traffic
type TrafficTracker interface {
	Finish(desc string, result interface{}, err error) bool
	Elapsed() (time.Duration, bool)
	Discard()
}

// A TrafficReportCallback is used to send a finished traffic report
type TrafficReportCallback func(name string, desc string, elapsed time.Duration, data map[string]string, result interface{}, err error)

// The DefaultTrafficTracker is used to track generic requests
type DefaultTrafficTracker struct {
	name     string
	data     map[string]string
	timeout  *time.Timer
	done     bool
	doneMux  sync.Mutex
	start    time.Time
	end      time.Time
	callback TrafficReportCallback
}

// NewDefaultTrafficTracker creates a new traffic tracker based on configuration
func NewDefaultTrafficTracker(name string, data map[string]string, timeout time.Duration, callback TrafficReportCallback) *DefaultTrafficTracker {
	dtt := &DefaultTrafficTracker{
		name:     name,
		data:     data,
		start:    time.Now(),
		callback: callback,
	}

	if timeout != time.Duration(0) {
		dtt.timeout = time.NewTimer(timeout)
		go func() {
			// wait on the timer
			<-dtt.timeout.C
			dtt.Finish(fmt.Sprintf("Timed out after %s", timeout.String()), nil, fmt.Errorf("Timeout"))
		}()
	}
	return dtt
}

// Finish closes a traffic tracking session
func (dtt *DefaultTrafficTracker) Finish(desc string, result interface{}, err error) bool {
	dtt.doneMux.Lock()
	defer dtt.doneMux.Unlock()
	if dtt.done {
		return false
	}
	dtt.done = true
	dtt.end = time.Now()
	elapsed, _ := dtt.Elapsed()
	// fire off the notification
	go dtt.callback(dtt.name, desc, elapsed, dtt.data, result, err)
	return true
}

// Discard throws away the current tracking data
func (dtt *DefaultTrafficTracker) Discard() {
	dtt.doneMux.Lock()
	defer dtt.doneMux.Unlock()

	// just mark done an don't call finish.
	dtt.done = true
}

// Elapsed returns the duration of traffic tracking
func (dtt *DefaultTrafficTracker) Elapsed() (time.Duration, bool) {
	if !dtt.done {
		return time.Duration(0), false
	}

	return dtt.end.Sub(dtt.start), true
}

// LoggingTrafficReporter logs traffic requests
type LoggingTrafficReporter struct {
	Prefix string
}

// Start begins logging traffic data
func (ltr LoggingTrafficReporter) Start(name string, data map[string]string, timeout time.Duration) TrafficTracker {
	// TODO(ai) this probably doesn't belong in the global logger
	logger := ulog.Logger()
	key := fmt.Sprintf("%s.%s", ltr.Prefix, name)
	return NewDefaultTrafficTracker(key, data, timeout,
		func(name string, desc string, elapsed time.Duration, data map[string]string, result interface{}, err error) {
			r := "OK"
			if err != nil {
				r = err.Error()
			}
			logger.Info(fmt.Sprintf("%s\t%dμs\t%s", name, elapsed.Nanoseconds()/1000, r))
		},
	)
}
