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
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	TrafficCorrelationID = "traffic_id"
	TrafficCallerName    = "traffic_cn"
)

type TrafficReporter interface {
	Start(name string, data map[string]string, timeout time.Duration) TrafficTracker
}

type TrafficTracker interface {
	Finish(desc string, result interface{}, err error) bool
	Elapsed() (time.Duration, bool)
	Discard()
}

type TrafficReportCallback func(name string, desc string, elapsed time.Duration, data map[string]string, result interface{}, err error)

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
			dtt.Finish(fmt.Sprintf("Timed out after %s", timeout.String()), nil, errors.New("Timeout"))
		}()
	}
	return dtt
}

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

func (dtt *DefaultTrafficTracker) Discard() {
	dtt.doneMux.Lock()
	defer dtt.doneMux.Unlock()

	// just mark done an don't call finish.
	dtt.done = true
}

func (dtt *DefaultTrafficTracker) Elapsed() (time.Duration, bool) {
	if !dtt.done {
		return time.Duration(0), false
	}

	return dtt.end.Sub(dtt.start), true
}

type LoggingTrafficReporter struct {
	Prefix string
}

func (ltr LoggingTrafficReporter) Start(name string, data map[string]string, timeout time.Duration) TrafficTracker {

	key := fmt.Sprintf("%s.%s", ltr.Prefix, name)
	return NewDefaultTrafficTracker(key, data, timeout,
		func(name string, desc string, elapsed time.Duration, data map[string]string, result interface{}, err error) {
			r := "OK"
			if err != nil {
				r = err.Error()
			}
			log.Output(0, fmt.Sprintf("%s\t%dμs\t%s", name, elapsed.Nanoseconds()/1000, r))
		},
	)
}
