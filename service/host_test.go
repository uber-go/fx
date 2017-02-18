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
	"testing"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

func TestOnCriticalError_NoObserver(t *testing.T) {
	err := errors.New("Blargh")
	sh := makeHost()
	control := sh.StartAsync()
	require.NoError(t, control.ServiceError, "Expected no error starting up")
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Server failed to start up after 1 second")
	case <-control.ReadyChan:
		// do nothing
	}
	go func() {
		<-control.ExitChan
	}()
	sh.OnCriticalError(err)
	assert.Equal(t, err, sh.shutdownReason.Error)
}

func TestSupportsRole_NoRoles(t *testing.T) {
	sh := &manager{}
	assert.True(t, sh.supportsRole("anything"), "Empty manager roles should pass any value")
}

func TestSuupportsRole_Matches(t *testing.T) {
	sh := &manager{
		roles: map[string]bool{"chilling": true},
	}
	assert.True(t, sh.supportsRole("chilling"), "Should support matching role")
}

func TestSupportsRole_NoMatch(t *testing.T) {
	sh := &manager{
		roles: map[string]bool{"business": true},
	}
	assert.False(t, sh.supportsRole("pleasure"), "Should not support non-matching role")
}

func TestHost_Modules(t *testing.T) {
	mods := []Module{}
	sh := &manager{modules: mods}

	copied := sh.Modules()
	assert.Equal(t, len(mods), len(copied), "Should have same amount of modules")
}

func TestTransitionState(t *testing.T) {
	sh := &manager{}
	observer := ObserverStub().(*StubObserver)
	require.NoError(t, WithObserver(observer)(sh))

	cases := []struct {
		name     string
		from, to State
	}{
		{
			name: "Uninitialized to Initialized",
			from: Uninitialized,
			to:   Initialized,
		},
		{
			name: "Uninitialized to Starting",
			from: Uninitialized,
			to:   Starting,
		},
		{
			name: "Initialized to Stopping",
			from: Initialized,
			to:   Stopping,
		},
		{
			name: "Running to stopped",
			from: Running,
			to:   Stopped,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sh.state = c.from
			sh.transitionState(c.to)
			assert.Equal(t, observer.state, c.to)
		})
	}
}

func TestLoadInstanceConfig_NoField(t *testing.T) {
	cfg := config.NewStaticProvider(nil)
	instance := struct{}{}

	assert.False(t, loadInstanceConfig(cfg, "anything", &instance), "No field defined on struct")
}

func TestLoadInstanceConfig_WithConfig(t *testing.T) {
	cfg := config.NewYAMLProviderFromBytes([]byte(`
foo:
  bar: 1
`))

	instance := struct {
		ServiceConfig struct {
			Bar int `yaml:"bar"`
		}
	}{}

	assert.True(t, loadInstanceConfig(cfg, "foo", &instance))
	assert.Equal(t, 1, instance.ServiceConfig.Bar)
}

func TestLoadInstanceConfig_IntKey(t *testing.T) {
	cfg := config.NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    1: baz
`))

	instance := struct {
		ServiceConfig struct {
			Bar map[int]interface{} `yaml:"bar"`
		}
	}{}
	assert.True(t, loadInstanceConfig(cfg, "foo", &instance))
	assert.Equal(t, instance.ServiceConfig.Bar[1], "baz")
}

func TestHostStop_NoError(t *testing.T) {
	sh := &manager{}
	assert.NoError(t, sh.Stop("testing", 1))
}

func TestOnCriticalError_ObserverShutdown(t *testing.T) {
	o := observerStub()
	sh := &manager{
		observer:    o,
		serviceCore: serviceCore{},
	}

	sh.OnCriticalError(errors.New("simulated shutdown"))
	assert.True(t, o.criticalError)
}

func TestShutdownWithError_ReturnsError(t *testing.T) {
	sh := makeRunningHost()
	exitCode := 1
	shutdown, err := sh.shutdown(errors.New("simulated"), "testing", &exitCode)
	assert.True(t, shutdown)
	assert.Error(t, err)
}

func TestHostShutdown_RunningService(t *testing.T) {
	sh := makeRunningHost()
	checkShutdown(t, sh, false)
}

func TestHostShutdown_CloseSuccessful(t *testing.T) {
	sh := makeRunningHost()
	sh.serviceCore.metricsCore = metricsCore{
		metrics:          tally.NoopScope,
		metricsCloser:    testutils.NopCloser{},
		runtimeCollector: metrics.NewRuntimeCollector(tally.NoopScope, time.Millisecond),
	}
	sh.serviceCore.tracerCore = tracerCore{
		tracerCloser: testutils.NopCloser{},
	}
	checkShutdown(t, sh, false)
}

func TestHostShutdown_MetricsCloserError(t *testing.T) {
	sh := makeRunningHost()
	sh.serviceCore.metricsCore = metricsCore{
		metrics:       tally.NoopScope,
		metricsCloser: testutils.ErrorCloser{},
	}
	checkShutdown(t, sh, true)
}

func TestHostShutdown_TracerCloserError(t *testing.T) {
	sh := makeRunningHost()
	sh.serviceCore.tracerCore = tracerCore{
		tracerCloser: testutils.ErrorCloser{},
	}
	checkShutdown(t, sh, true)
}

func checkShutdown(t *testing.T, h *manager, expectedErr bool) {
	exitCode := 1
	shutdown, err := h.shutdown(nil, "testing", &exitCode)
	assert.True(t, shutdown)
	if expectedErr {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func TestHostStart_InShutdown(t *testing.T) {
	sh := &manager{
		inShutdown: true,
	}
	control := sh.StartAsync()
	assert.Error(t, control.ServiceError)
}

func TestHostStart_AlreadyRunning(t *testing.T) {
	sh := makeRunningHost()
	control := sh.StartAsync()
	assert.Error(t, control.ServiceError)
}

func TestStartWithObserver_InitError(t *testing.T) {
	obs := observerStub()
	obs.initError = errors.New("can't touch this")
	sh := &manager{
		observer: obs,
	}
	control := sh.StartAsync()
	assert.Error(t, control.ServiceError)
	assert.True(t, obs.init)
}

func TestAddModule_Locked(t *testing.T) {
	sh := &manager{
		locked: true,
	}
	assert.Error(t, sh.addModule(nil))
}

func TestAddModule_NotLocked(t *testing.T) {
	sh := &manager{}
	mod := NewStubModule(sh)
	assert.NoError(t, sh.addModule(mod))
	assert.Equal(t, sh, mod.Host)
}

func TestStartStopRegressionDeadlock(t *testing.T) {
	// TODO(glib): sort out this test
	t.Skip("Fix me when Start/Stop functions are refactored")
	sh := makeHost()
	go func() {
		time.Sleep(50 * time.Millisecond)
		sh.Stop("stop nao!", 1)
	}()
	sh.Start()
}

func TestStartModule_NoErrors(t *testing.T) {
	s := makeHost()
	mod := NewStubModule(s)
	require.NoError(t, s.addModule(mod))

	control := s.StartAsync()
	go func() {
		<-control.ExitChan
	}()
	defer func() {
		assert.NoError(t, s.Stop("test", 0))
		assert.Equal(t, s.state, Stopped)
	}()

	assert.NoError(t, control.ServiceError)
	assert.True(t, mod.IsRunning())
	assert.Equal(t, s.state, Running)
}

func TestStartHost_WithErrors(t *testing.T) {
	s := makeHost()
	mod := NewStubModule(s)
	mod.StartError = errors.New("can't start this")
	require.NoError(t, s.addModule(mod))

	control := s.StartAsync()
	go func() {
		<-control.ExitChan
	}()
	assert.Error(t, control.ServiceError)
}

func makeRunningHost() *manager {
	h := makeHost()
	h.closeChan = make(chan Exit, 1) // Indicates service is running
	return h
}

func makeHost() *manager {
	return &manager{
		serviceCore: serviceCore{},
	}
}
