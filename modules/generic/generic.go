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
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

// Module is a simpliciation of the Module interface
// that can be wrapped with a Module for easier implemenation.
type Module interface {
	// This will be called after ModuleBase, log, and config are populated.
	// If the module wishes, these can be stored for use in the module.
	// Start and stop should not be called before this is called.
	Initialize(moduleBase modules.ModuleBase, log ulog.Log, config interface{}) error
	// Start the module, on return the module is expected to be started
	// If there is an error, the module is not expected to be started.
	Start() error
	// Stop the module, on return the module is expected to be stopped
	// If there is an error, the module is still expected to be stopped.
	Stop() error
}

// NewModule returns a ModuleCreateFunc for the given GenericModule.
//
// config should be a struct pointer to a configuration struct.
//
//   type FooConfig struct {
//       modules.ModuleConfig
//       Foo string
//   }
//
//   NewModule("foo", module, &FooConfig{})
func NewModule(
	moduleName string,
	module Module,
	config interface{},
	options ...modules.Option,
) service.ModuleCreateFunc {
	return func(moduleCreateInfo service.ModuleCreateInfo) ([]service.Module, error) {
		module, err := newWrapperModule(moduleCreateInfo, moduleName, module, config, options...)
		if err != nil {
			return nil, err
		}
		return []service.Module{module}, nil
	}
}

type wrapperModule struct {
	moduleName string
	module     Module
	lock       sync.RWMutex
	running    bool
}

func newWrapperModule(
	moduleCreateInfo service.ModuleCreateInfo,
	moduleName string,
	module Module,
	config interface{},
	options ...modules.Option,
) (service.Module, error) {
	for _, option := range options {
		if err := option(&moduleCreateInfo); err != nil {
			return nil, err
		}
	}
	if moduleCreateInfo.Name != "" {
		moduleName = moduleCreateInfo.Name
	}
	moduleBase := *modules.NewModuleBase(
		moduleName,
		moduleCreateInfo.Host,
		moduleCreateInfo.Roles,
	)
	if err := moduleBase.Host().Config().Scope("modules").Get(moduleName).PopulateStruct(config); err != nil {
		return nil, err
	}
	if err := module.Initialize(
		moduleBase,
		ulog.Logger().With("moduleName", moduleName),
		config,
	); err != nil {
		return nil, err
	}
	return &wrapperModule{moduleName: moduleName, module: module}, nil
}

func (m *wrapperModule) Name() string {
	return m.moduleName
}

func (m *wrapperModule) Start(readyC chan<- struct{}) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()
	errC := make(chan error, 1)
	if err := m.module.Start(); err != nil {
		errC <- err
		return errC
	}
	errC <- nil
	m.running = true
	readyC <- struct{}{}
	return errC
}

func (m *wrapperModule) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.running = false
	return m.module.Stop()
}

func (m *wrapperModule) IsRunning() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.running
}
