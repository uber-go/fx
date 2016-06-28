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

func (dtt DefaultTrafficTracker) Elapsed() (time.Duration, bool) {
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
