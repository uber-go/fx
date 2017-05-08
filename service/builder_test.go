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

	. "go.uber.org/fx/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/config"
)

var (
	nopModuleProvider    = &StubModuleProvider{"nop", nopModule}
	errModuleProvider    = &StubModuleProvider{"err", errModule}
	startTimeoutProvider = &StubModuleProvider{"timeoutStart", timeoutStartModule}
	stopTimeoutProvider  = &StubModuleProvider{"timeoutStop", timeoutStopModule}
)

func TestWithModules_OK(t *testing.T) {
	_, err := WithModule(nopModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(nil)),
	).Build()
	assert.NoError(t, err)
}

func TestWithModules_Err(t *testing.T) {
	_, err := WithModule(errModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(nil)),
	).Build()
	assert.Error(t, err)
}

func TestWithModules_SkipsModulesBadInit(t *testing.T) {
	empty := ""
	_, err := WithModule(nopModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(&empty)),
	).Build()
	assert.Error(t, err, "Expected service name to be provided")
}

func TestWithModules_StartTimeout(t *testing.T) {
	cfg := config.NewStaticProvider(map[string]interface{}{
		"startTimeout": time.Microsecond,
		"stopTimeout":  time.Microsecond,
		"name":         "test",
		"owner":        "test@uber.com",
	})

	svc, err := WithModule(startTimeoutProvider).
		WithModule(startTimeoutProvider).
		WithOptions(
			WithConfiguration(cfg),
		).Build()

	require.NoError(t, err)

	ctl := svc.StartAsync()
	require.Error(t, ctl.ServiceError)
	assert.Contains(t, ctl.ServiceError.Error(), "timeoutStart")
	assert.Contains(t, ctl.ServiceError.Error(), `didn't start after "1µs"`)
}

func TestWithModules_StopTimeout(t *testing.T) {
	cfg := config.NewStaticProvider(map[string]interface{}{
		"startTimeout": time.Microsecond,
		"stopTimeout":  time.Microsecond,
		"name":         "test",
		"owner":        "test@uber.com",
	})

	svc, err := WithModule(stopTimeoutProvider).
		WithModule(stopTimeoutProvider).
		WithOptions(
			WithConfiguration(cfg),
		).Build()

	require.NoError(t, err)

	ctl := svc.StartAsync()
	require.NoError(t, ctl.ServiceError)

	err = svc.Stop("someReason", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeoutStop")
	assert.Contains(t, err.Error(), `timedout after "1µs"`)
}

func TestDefaultTimeouts(t *testing.T) {
	svc, err := WithModule(stopTimeoutProvider).
		WithModule(stopTimeoutProvider).
		WithOptions(
			WithConfiguration(StaticAppData(nil)),
		).Build()

	require.NoError(t, err)
	m, ok := svc.(*manager)
	require.True(t, ok, "expect manager returned by Build")
	require.NotNil(t, m)
	assert.Equal(t, 10*time.Second, m.StartTimeout)
	assert.Equal(t, 10*time.Second, m.StopTimeout)
}

func nopModule() (Module, error) {
	return nil, nil
}

func errModule() (Module, error) {
	return nil, errors.New("intentional module creation failure")
}

func timeoutStartModule() (Module, error) {
	return timeoutStart{}, nil
}

type timeoutStart struct{}

func (timeoutStart) Start() error {
	<-make(chan int)
	return nil
}

func (timeoutStart) Stop() error {
	return nil
}

func timeoutStopModule() (Module, error) {
	return timeoutStop{}, nil
}

type timeoutStop struct{}

func (timeoutStop) Start() error {
	return nil
}

func (timeoutStop) Stop() error {
	<-make(chan int)
	return nil
}
