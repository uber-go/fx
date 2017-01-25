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

package task

import (
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"github.com/pkg/errors"
)

// ModuleType represents the task module type
const ModuleType = "task"

type globalBackend struct {
	backend Backend
	sync.RWMutex
}

var (
	_backendRegisterFn backendRegisterFn
	_globalBackendMu   sync.RWMutex
	_globalBackend     Backend = &NopBackend{}
)

// GlobalBackend returns global instance of the backend
// TODO (madhu): Make work with multiple backends
func GlobalBackend() Backend {
	_globalBackendMu.RLock()
	defer _globalBackendMu.RUnlock()
	return _globalBackend
}

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

func newAsyncModule(mi service.ModuleCreateInfo) (service.Module, error) {
	backend, err := _backendRegisterFn(mi.Host, Config{})
	if err != nil {
		return nil, err
	}
	_globalBackendMu.Lock()
	_globalBackend = backend
	_globalBackendMu.Unlock()
	return &AsyncModule{
		Backend: backend,
		modBase: *modules.NewModuleBase(ModuleType, "task", mi.Host, []string{}),
	}, nil
}

// AsyncModule denotes the asynchronous task queue module
type AsyncModule struct {
	Backend
	modBase modules.ModuleBase
	config  Config
}

// Config contains config for task backends
type Config struct {
	broker  string `yaml:"broker"`
	timeout string `yaml:"timeout"`
}

type backendRegisterFn func(service.Host, Config) (Backend, error)

// RegisterAsyncBackend registers the backend for the async task module
func RegisterAsyncBackend(backendRegisterFn backendRegisterFn) {
	_backendRegisterFn = backendRegisterFn
}
