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

	"github.com/uber-go/tally"

	"go.uber.org/fx/service"
)

var (
	_globalBackendMu          sync.RWMutex
	_globalBackend            Backend = &NopBackend{}
	_globalBackendStatsClient         = newStatsClient(tally.NoopScope)
	_asyncMod                 service.Module
	_asyncModErr              error
	_once                     sync.Once
)

// GlobalBackend returns global instance of the backend
// TODO (madhu): Make work with multiple backends
func GlobalBackend() Backend {
	_globalBackendMu.RLock()
	defer _globalBackendMu.RUnlock()
	return _globalBackend
}

func globalBackendStatsClient() *statsClient {
	_globalBackendMu.RLock()
	defer _globalBackendMu.RUnlock()
	return _globalBackendStatsClient
}

// NewModule creates an async task queue module
func NewModule(createFunc BackendCreateFunc) service.ModuleCreateFunc {
	return func(mi service.Host) (service.Module, error) {
		return newAsyncModuleSingleton(mi, createFunc)
	}
}

func newAsyncModuleSingleton(
	mi service.Host,
	createFunc BackendCreateFunc,
) (service.Module, error) {
	_once.Do(func() {
		_asyncMod, _asyncModErr = newAsyncModule(mi, createFunc)
	})
	return _asyncMod, _asyncModErr
}

func newAsyncModule(
	mi service.Host,
	createFunc BackendCreateFunc,
) (service.Module, error) {
	backend, err := createFunc(mi)
	if err != nil {
		return nil, err
	}
	_globalBackendMu.Lock()
	_globalBackend = backend
	_globalBackendStatsClient = newStatsClient(mi.Metrics())
	_globalBackendMu.Unlock()
	return backend, nil
}

// BackendCreateFunc creates a backend implementation
type BackendCreateFunc func(host service.Host) (Backend, error)
