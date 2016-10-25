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

package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/fx/core/config"

	"github.com/stretchr/testify/assert"
)

type testStatsReporter struct {
	m sync.Mutex

	cw sync.WaitGroup
	gw sync.WaitGroup
	tw sync.WaitGroup

	counters map[string]int64
	gauges   map[string]int64
	timers   map[string]time.Duration
}

func (r *testStatsReporter) ReportCounter(name string, tags map[string]string, value int64) {
	r.m.Lock()
	r.counters[name] += value
	r.cw.Done()
	r.m.Unlock()
}

func (r *testStatsReporter) ReportGauge(name string, tags map[string]string, value int64) {
	r.m.Lock()
	r.gauges[name] = value
	r.gw.Done()
	r.m.Unlock()
}

func (r *testStatsReporter) ReportTimer(name string, tags map[string]string, interval time.Duration) {
	r.m.Lock()
	r.timers[name] = interval
	r.tw.Done()
	r.m.Unlock()
}

func (r *testStatsReporter) Flush() {}

func newTestStatsReporter() *testStatsReporter {
	return &testStatsReporter{
		counters: make(map[string]int64),
		gauges:   make(map[string]int64),
		timers:   make(map[string]time.Duration),
	}
}

func TestServiceCreation(t *testing.T) {
	r := newTestStatsReporter()
	r.cw.Add(1)
	svc := New(
		WithStatsReporter(r, 50*time.Millisecond),
	)
	assert.NotNil(t, svc, "Service should be created")
	r.cw.Wait()

	assert.Equal(t, r.counters["boot"], int64(1))
}

func TestWithObserver_Nil(t *testing.T) {
	svc := New(
		WithObserver(nil),
	)
	assert.Nil(t, svc.Observer(), "Observer should be nil")
}

func TestServiceWithRoles(t *testing.T) {
	defer withConfigData(map[string]interface{}{
		"applicationID":    "name",
		"applicationOwner": "owner",
		"roles.0":          "foo",
	})()

	svc := New()
	assert.Contains(t, svc.Roles(), "foo")
}

func TestBadOption_Panics(t *testing.T) {
	defer withConfigData(nil)()
	opt := func(_ Host) error {
		return errors.New("nope")
	}

	assert.Panics(t, func() {
		New(opt)
	})
}

func TestNew_WithObserver(t *testing.T) {
	defer withConfigData(nil)()
	o := observerStub()
	svc := New(WithObserver(o))
	assert.Equal(t, o, svc.Observer())
}

// withConfigData sets a global config and returns a function to defer reset
func withConfigData(data map[string]interface{}) func() {
	config.RegisterProviders(config.StaticProvider(data))
	config.InitializeGlobalConfig()
	return config.ResetGlobal
}
