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
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
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
	scope := tally.NewRootScope("", nil, r, 50*time.Millisecond)
	svc, err := New(
		withConfig(validServiceConfig),
		WithMetricsRootScope(scope),
	)
	require.NoError(t, err)
	assert.NotNil(t, svc, "Service should be created")
	r.cw.Wait()

	assert.Equal(t, r.counters["boot"], int64(1))
}

func TestWithObserver_Nil(t *testing.T) {
	svc, err := New(
		withConfig(validServiceConfig),
		WithObserver(nil),
	)
	require.NoError(t, err)

	assert.Nil(t, svc.Observer(), "Observer should be nil")
}

func TestServiceCreation_MissingRequiredParams(t *testing.T) {
	_, err := New(withConfig(nil))
	assert.Error(t, err, "should fail with missing service name")
	assert.Contains(t, err.Error(), "zero value")
}

func TestServiceWithRoles(t *testing.T) {
	data := map[string]interface{}{
		"applicationID":    "name",
		"applicationOwner": "owner",
		"roles.0":          "foo",
	}
	cfgOpt := withConfig(data)

	svc, err := New(cfgOpt)
	require.NoError(t, err)

	assert.Contains(t, svc.Roles(), "foo")
}

func TestBadOption_Panics(t *testing.T) {
	opt := func(_ Host) error {
		return errors.New("nope")
	}

	_, err := New(withConfig(validServiceConfig), opt)
	assert.Error(t, err, "should fail with invalid option")
}

func TestNew_WithObserver(t *testing.T) {
	o := observerStub()
	svc, err := New(withConfig(validServiceConfig), WithObserver(o))
	require.NoError(t, err)
	assert.Equal(t, o, svc.Observer())
}

var validServiceConfig = map[string]interface{}{
	"applicationID":    "test-suite",
	"applicationOwner": "go.uber.org/fx",
}

func withConfig(data map[string]interface{}) Option {
	return WithConfiguration(config.NewStaticProvider(data))
}
