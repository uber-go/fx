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
	"testing"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnCriticalError_NoObserver(t *testing.T) {
	err := errors.New("Blargh")
	sh := makeHost()
	closeCh, ready, err := sh.Start(false)
	require.NoError(t, err, "Expected no error starting up")
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Server failed to start up after 1 second")
	case <-ready:
		// do nothing
	}
	go func() {
		<-closeCh
	}()
	sh.OnCriticalError(err)
	assert.Equal(t, err, sh.shutdownReason.Error)
}

func TestSupportsRole_NoRoles(t *testing.T) {
	sh := &host{}
	assert.True(t, sh.supportsRole("anything"), "Empty host roles should pass any value")
}

func TestSuupportsRole_Matches(t *testing.T) {
	sh := &host{
		roles: map[string]bool{"chilling": true},
	}
	assert.True(t, sh.supportsRole("chilling"), "Should support matching role")
}

func TestSupportsRole_NoMatch(t *testing.T) {
	sh := &host{
		roles: map[string]bool{"business": true},
	}
	assert.False(t, sh.supportsRole("pleasure"), "Should not support non-matching role")
}

func TestHost_Modules(t *testing.T) {
	mods := []Module{}
	sh := &host{modules: mods}

	copied := sh.Modules()
	assert.Equal(t, len(mods), len(copied), "Should have same amount of modules")
}

func TestTransitionState(t *testing.T) {
	sh := &host{}
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
	sh := &host{}
	assert.NoError(t, sh.Stop("testing", 1))
}

func TestOnCriticalError_ObserverShutdown(t *testing.T) {
	o := observerStub()
	sh := &host{
		observer: o,
		serviceCore: serviceCore{
			log: ulog.NoopLogger,
		},
	}

	sh.OnCriticalError(errors.New("simulated shutdown"))
	assert.True(t, o.criticalError)
}

func TestShutdownWithError_ReturnsError(t *testing.T) {
	sh := &host{
		closeChan: make(chan Exit, 1),
	}
	exitCode := 1
	shutdown, err := sh.shutdown(errors.New("simulated"), "testing", &exitCode)
	assert.True(t, shutdown)
	assert.Error(t, err)
}

func TestHostStart_InShutdown(t *testing.T) {
	sh := &host{
		inShutdown: true,
	}
	_, _, err := sh.Start(false)
	assert.Error(t, err)
}

func TestHostStart_AlreadyRunning(t *testing.T) {
	sh := &host{
		closeChan: make(chan Exit, 1),
	}
	_, _, err := sh.Start(false)
	assert.NoError(t, err)
}

func TestStartWithObserver_InitError(t *testing.T) {
	obs := observerStub()
	obs.initError = errors.New("can't touch this")
	sh := &host{
		observer: obs,
	}
	_, _, err := sh.Start(false)
	assert.Error(t, err)
	assert.True(t, obs.init)
}

func TestAddModule_Locked(t *testing.T) {
	sh := &host{
		locked: true,
	}
	assert.Error(t, sh.addModule(nil))
}

func TestAddModule_NotLocked(t *testing.T) {
	mod := NewStubModule()
	sh := &host{}
	assert.NoError(t, sh.addModule(mod))
	assert.Equal(t, sh, mod.Host)
}

func TestStartModule_NoErrors(t *testing.T) {
	s := makeHost()
	mod := NewStubModule()
	require.NoError(t, s.addModule(mod))

	closeCh, _, err := s.Start(false)
	go func() {
		<-closeCh
	}()
	defer func() {
		assert.NoError(t, s.Stop("test", 0))
		assert.Equal(t, s.state, Stopped)
	}()

	assert.NoError(t, err)
	assert.True(t, mod.IsRunning())
	assert.Equal(t, s.state, Running)
}

func TestStartHost_WithErrors(t *testing.T) {
	s := makeHost()
	mod := NewStubModule()
	mod.StartError = errors.New("can't start this")
	require.NoError(t, s.addModule(mod))

	closeCh, _, err := s.Start(false)
	go func() {
		<-closeCh
	}()
	assert.Error(t, err)
}

func makeHost() *host {
	return &host{
		serviceCore: serviceCore{
			log: ulog.NoopLogger,
		},
	}
}
