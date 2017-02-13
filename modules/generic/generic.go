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

	"github.com/opentracing/opentracing-go"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

// PopulateStruct populates the given configuration struct using the given Controller.
func PopulateStruct(controller Controller, config interface{}) error {
	return controller.Host().Config().Scope("modules").Get(controller.Name()).PopulateStruct(config)
}

// Controller holds data for a Module, and provides functionality
// for the Module to notify the wrapping service.Module of state changes.
// TODO(pedge): not the best name
// TODO(pedge): split up into read-only and read-write interfaces?
type Controller interface {
	Host() service.Host
	Roles() []string
	Name() string
	Tracer() opentracing.Tracer
	Log() ulog.Log
	// NotifyStopped will result in the wrapping service.Module's
	// IsRunning() method to return false.
	NotifyStopped()
}

// Module is a simpliciation of the service.Module interface
// that can be wrapped with a service.Module for easier implemenation.
type Module interface {
	// This will be called after the Controller is populated.
	// If the module wishes, this can be stored for use in the module.
	//
	// This should only be called by the generic package, and unless there are
	// other calls, it can be safely assumed that this will be called exactly once,
	// and called before Start() and Stop() are ever called.
	Initialize(controller Controller) error
	// Start the module.
	// On return,  the module is expected to be started, unless there is an error.
	Start() error
	// Stop the module.
	// On return the module is expected to be stopped, regardless of if there is an error.
	Stop() error
}

// NewModule returns a ModuleCreateFunc for the given Module.
func NewModule(
	moduleName string,
	module Module,
	options ...modules.Option,
) service.ModuleCreateFunc {
	return func(moduleCreateInfo service.ModuleCreateInfo) ([]service.Module, error) {
		module, err := newWrapperModule(moduleCreateInfo, moduleName, module, options...)
		if err != nil {
			return nil, err
		}
		return []service.Module{module}, nil
	}
}

type wrapperModule struct {
	modules.ModuleBase
	log        ulog.Log
	moduleName string
	module     Module
	lock       sync.RWMutex
	running    bool
}

func newWrapperModule(
	moduleCreateInfo service.ModuleCreateInfo,
	moduleName string,
	module Module,
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
	wrapperModule := &wrapperModule{
		ModuleBase: moduleBase,
		log:        ulog.Logger().With("moduleName", moduleName),
		moduleName: moduleName,
		module:     module,
	}
	if err := module.Initialize(wrapperModule); err != nil {
		return nil, err
	}
	return wrapperModule, nil
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

func (m *wrapperModule) Log() ulog.Log {
	return m.log
}

func (m *wrapperModule) NotifyStopped() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.running = false
}
