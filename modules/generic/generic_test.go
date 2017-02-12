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
	testModule, _ := setup(t, testConfig, "test")
	assert.Equal(t, testConfig.Test, testModule.config.Test)
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
	return NewModule(moduleName, testModule, &testConfig{}, options...)
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
	config *testConfig
}

func newTestModule() *testModule {
	return &testModule{}
}

func (m *testModule) Initialize(controller Controller, config interface{}) error {
	m.Controller = controller
	m.config = config.(*testConfig)
	return nil
}

func (m *testModule) Start() error {
	return nil
}

func (m *testModule) Stop() error {
	return nil
}
