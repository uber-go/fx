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
)

type globalBackend struct {
	backend Backend
	sync.RWMutex
}

var (
	_globalBackendMu sync.RWMutex
	_globalBackend   Backend = &NopBackend{}
	_asyncMod        service.Module
	_once            sync.Once
)

// GlobalBackend returns global instance of the backend
// TODO (madhu): Make work with multiple backends
func GlobalBackend() Backend {
	_globalBackendMu.RLock()
	defer _globalBackendMu.RUnlock()
	return _globalBackend
}

// NewModule creates an async task queue module
func NewModule(backend Backend) service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod := newAsyncModuleSingleton(mi, backend)
		return []service.Module{mod}, nil
	}
}

func newAsyncModuleSingleton(mi service.ModuleCreateInfo, backend Backend) service.Module {
	_once.Do(func() {
		_asyncMod = newAsyncModule(mi, backend)
	})
	return _asyncMod
}

func newAsyncModule(mi service.ModuleCreateInfo, backend Backend) service.Module {
	_globalBackendMu.Lock()
	_globalBackend = backend
	_globalBackendMu.Unlock()
	return &AsyncModule{
		Backend: backend,
		modBase: *modules.NewModuleBase("task", mi.Host, []string{}),
	}
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
