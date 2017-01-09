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

package utask

import (
	"sync"

	"github.com/pkg/errors"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

// ModuleType represents the utask module type
const ModuleType = "utask"

var _backendRegisterFn backendRegisterFn

// NewModule creates an async task queue module
func NewModule() service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod, err := newAsyncModule(mi)
		if err != nil {
			return nil, errors.Wrap(err, "unable to instantiate async task module")
		}
		return []service.Module{mod}, nil
	}
}

func newAsyncModule(mi service.ModuleCreateInfo) (*AsyncModule, error) {
	backendMod, err := _backendRegisterFn(mi.Host, Config{})
	if err != nil {
		return nil, err
	}
	return &AsyncModule{
		ModuleBase:    *modules.NewModuleBase(ModuleType, "task", mi.Host, []string{}),
		backendModule: backendMod,
	}, nil
}

// AsyncModule denotes the asynchronous task queue module
type AsyncModule struct {
	modules.ModuleBase
	stateMu       sync.RWMutex
	config        Config
	backendModule service.Module
}

// Config contains config for task backends
type Config struct {
	broker  string `yaml:"broker"`
	timeout string `yaml:"timeout"`
}

// Start begins serving requests over the module
func (m *AsyncModule) Start(readyCh chan<- struct{}) <-chan error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return nil
}

// Stop shuts down the module
func (m *AsyncModule) Stop() error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return nil
}

// IsRunning returns whether the module is running
func (m *AsyncModule) IsRunning() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return false
}

type backendRegisterFn func(service.Host, Config) (service.Module, error)

// RegisterAsyncBackend registers the backend for the async task module
func RegisterAsyncBackend(backendRegisterFn backendRegisterFn) {
	_backendRegisterFn = backendRegisterFn
}
