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

	"golang.org/x/net/context"

	"github.com/pkg/errors"
	"github.com/uber-go/tally"

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"
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

// New creates an async task queue ModuleProvider.
func New(createFunc BackendCreateFunc, options ...ModuleOption) service.ModuleProvider {
	return service.ModuleProviderFromFunc("task", func(host service.Host) (service.Module, error) {
		return newAsyncModuleSingleton(host, createFunc, options...)
	})
}

func newAsyncModuleSingleton(
	host service.Host,
	createFunc BackendCreateFunc,
	options ...ModuleOption,
) (service.Module, error) {
	_once.Do(func() {
		_asyncMod, _asyncModErr = newAsyncModule(host, createFunc, options...)
	})
	return _asyncMod, _asyncModErr
}

func newAsyncModule(
	host service.Host,
	createFunc BackendCreateFunc,
	options ...ModuleOption,
) (service.Module, error) {
	config := &Config{}
	for _, option := range options {
		if err := option(config); err != nil {
			return nil, err
		}
	}
	b, err := createFunc(host)
	if err != nil {
		return nil, err
	}
	mBackend := &managedBackend{b, *config}
	_globalBackendMu.Lock()
	_globalBackend = mBackend
	_globalBackendStatsClient = newStatsClient(host.Metrics())
	_globalBackendMu.Unlock()
	return mBackend, nil
}

// managedBackend is the root for all backends and controls execution
type managedBackend struct {
	Backend
	config Config
}

// Start implements the Module interface
func (b *managedBackend) Start() error {
	if err := b.Backend.Start(); err != nil {
		return errors.Wrap(err, "unable to start backend")
	}
	if !b.config.DisableExecution {
		errorCh := b.ExecuteAsync()
		go func() {
			for err := range errorCh {
				ulog.Logger(context.Background()).Error("task execute error", zap.Error(err))
			}
		}()
	}
	return nil
}

// BackendCreateFunc creates a backend implementation
type BackendCreateFunc func(service.Host) (Backend, error)

// ModuleOption is a function that configures module creation.
type ModuleOption func(*Config) error

// Config represents the options for the task module
type Config struct {
	DisableExecution bool
}

// DisableExecution disables task execution and only allows enqueuing
func DisableExecution() ModuleOption {
	return func(config *Config) error {
		config.DisableExecution = true
		return nil
	}
}
