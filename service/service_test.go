// Copyright (c) 2017 Uber Technologies, Inc.
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

	"go.uber.org/fx/config"
	"go.uber.org/fx/testutils/metrics"
	"go.uber.org/fx/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

func TestServiceCreation(t *testing.T) {
	r := metrics.NewTestStatsReporter()
	r.CountersWG.Add(1)
	scope, closer := tally.NewRootScope("", nil, r, 50*time.Millisecond, tally.DefaultSeparator)
	defer closer.Close()
	svc, err := New(
		withConfig(validServiceConfig),
		WithMetrics(scope, nil),
	)
	require.NoError(t, err)
	assert.NotNil(t, svc, "Service should be created")
	r.CountersWG.Wait()

	assert.Equal(t, r.Counters["boot"], int64(1))
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
		"name":    "name",
		"owner":   "owner",
		"roles.0": "foo",
	}
	cfgOpt := withConfig(data)

	svc, err := New(cfgOpt)
	require.NoError(t, err)

	assert.Contains(t, svc.Roles(), "foo")
}

func TestServiceWithMetricsRuntime(t *testing.T) {
	data := []byte(`
name: name
owner: owner
metrics:
  runtime:
    disabled: true
`)
	cfgOpt := WithConfiguration(config.NewYAMLProviderFromBytes(data))

	svc, err := New(cfgOpt)
	require.NoError(t, err)

	assert.Nil(t, svc.RuntimeMetricsCollector())
}

func TestServiceWithSentryHook(t *testing.T) {
	data := []byte(`
name: name
owner: owner
logging:
  sentry:
    dsn: http://user:secret@your.sentry.dsn/project
`)
	cfgOpt := WithConfiguration(config.NewYAMLProviderFromBytes(data))

	svc, err := New(cfgOpt)
	require.NoError(t, err)
	assert.NotNil(t, svc.(*host).log)
	// Note: Sentry is not accessible so we cannot directly test it here. Just invoking the code
	// path to make sure there is no panic
	svc.(*host).log.Info("Testing sentry call")
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
	"name":  "test-suite",
	"owner": "go.uber.org/fx",
}

func withConfig(data map[string]interface{}) Option {
	return WithConfiguration(config.NewStaticProvider(data))
}

func TestAfterStartObserver(t *testing.T) {
	t.Parallel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	h := &host{
		serviceCore: serviceCore{
			loggingCore: loggingCore{
				log: ulog.NopLogger,
			},
		},
		observer: AfterStart(func() {
			wg.Done()
		}),
	}

	h.StartAsync()
	wg.Wait()
}
