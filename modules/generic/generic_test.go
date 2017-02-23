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

package generic

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/fx/config"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

func TestConfig(t *testing.T) {
	testConfig := &testConfig{Test: "foo"}
	testModule, _ := setup(t, testConfig, "bar")
	assert.Equal(t, "foo", testModule.config.Test)
}

func TestName(t *testing.T) {
	_, module := setup(t, nil, "foo")
	assert.Equal(t, "foo", module.Name())
}

func TestStartStop(t *testing.T) {
	testModule, module := setup(t, nil, "foo")
	errC := module.Start(make(chan struct{}, 1))
	assert.NoError(t, <-errC)
	assert.Equal(t, 1, testModule.startCount)
	assert.True(t, module.IsRunning())
	assert.NoError(t, module.Stop())
	assert.Equal(t, 1, testModule.stopCount)
	assert.False(t, module.IsRunning())
}

func TestStartError(t *testing.T) {
	testModule, module := setup(t, nil, "foo")
	testModule.err = errors.New("error")
	errC := module.Start(make(chan struct{}, 1))
	assert.Error(t, <-errC)
}

func TestNotifyStopped(t *testing.T) {
	testModule, module := setup(t, nil, "foo")
	_ = module.Start(make(chan struct{}, 1))
	testModule.NotifyStopped()
	assert.False(t, module.IsRunning())
}

func setup(
	t *testing.T,
	testConfig *testConfig,
	moduleName string,
) (*testModule, service.Module) {
	testModule := newTestModule()
	module, err := newModule(testModule, testConfig, moduleName)
	require.NoError(t, err)
	return testModule, module
}

func newModule(
	testModule *testModule,
	testConfig *testConfig,
	moduleName string,
) (service.Module, error) {
	modules, err := newModuleFunc(moduleName, testModule)(
		service.ModuleCreateInfo{
			Name: moduleName,
			Host: service.NopHostWithConfig(
				newConfigProvider(
					moduleName,
					testConfig,
				),
			),
		},
	)
	if err != nil {
		return nil, err
	}
	return modules[0], nil
}

func newModuleFunc(moduleName string, testModule *testModule, options ...modules.Option) service.ModuleCreateFunc {
	return NewModule(moduleName, testModule, options...)
}

func newConfigProvider(moduleName string, testConfig *testConfig) config.Provider {
	if testConfig == nil {
		return nil
	}
	return config.NewYAMLProviderFromBytes(newYAMLConfigBytes(moduleName, testConfig))
}

func newYAMLConfigBytes(moduleName string, testConfig *testConfig) []byte {
	return []byte(fmt.Sprintf(`
modules:
  %s:
    test: %s
`, moduleName, testConfig.Test))
}

type testConfig struct {
	Test string `yaml:"test"`
}

type testModule struct {
	Controller
	config     testConfig
	startCount int
	stopCount  int
	err        error
}

func newTestModule() *testModule {
	return &testModule{}
}

func (m *testModule) Initialize(controller Controller) error {
	m.Controller = controller
	return PopulateStruct(controller, &m.config)
}

func (m *testModule) Start() error {
	m.startCount++
	return m.err
}

func (m *testModule) Stop() error {
	m.stopCount++
	return m.err
}
