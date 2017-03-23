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
	"context"
	"errors"
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_nopBackend       = &NopBackend{}
	_nopBackendFn     = func(service.Host) (Backend, error) { return _nopBackend, nil }
	_memBackendFn     = func(host service.Host) (Backend, error) { return NewInMemBackend(host), nil }
	_backendFnWithErr = func(service.Host) (Backend, error) { return nil, errors.New("bknd err") }
	_errBackendFn     = func(service.Host) (Backend, error) { return errBackend{*_nopBackend}, nil }
)

type errBackend struct{ NopBackend }

func (b errBackend) ExecuteAsync() error {
	return errors.New("execute async error")
}

func TestNew(t *testing.T) {
	b := NewInMemBackend(newTestHost(t))
	bFn := func(host service.Host) (Backend, error) { return b, nil }
	mod := createModule(t, bFn) // Singleton modules get saved
	require.NoError(t, mod.Start())
	require.Equal(t, b, mod.(*managedBackend).Backend)
	mod = createModule(t, _nopBackendFn) // Singleton returns nop even though mem backend is input
	require.Equal(t, b, mod.(*managedBackend).Backend)
}

func TestMemBackendModuleWorkflowWithContext(t *testing.T) {
	mod, err := newAsyncModule(newTestHost(t), _memBackendFn, DisableExecution())
	require.NoError(t, err)
	require.NotNil(t, mod)
	b := GlobalBackend()
	require.True(t, b.(*managedBackend).config.DisableExecution)
	require.NoError(t, b.Start())
	fn := func(ctx context.Context) error {
		return errors.New("hello error")
	}
	require.NoError(t, b.ExecuteAsync()) // This is required module is set with DisableExecution
	require.NoError(t, Register(fn))
	require.NoError(t, Enqueue(fn, context.Background()))
	errorCh := b.(*managedBackend).Backend.(*inMemBackend).errorCh
	require.Error(t, <-errorCh)
}

func TestModuleStartWithExecuteAsyncError(t *testing.T) {
	mod, err := newAsyncModule(newTestHost(t), _errBackendFn)
	require.NoError(t, err)
	require.NotNil(t, mod)
	err = mod.Start()
	require.Error(t, err)
	require.Contains(t, err.Error(), "execute async error")
}

func TestNewError(t *testing.T) {
	mod, err := newAsyncModule(newTestHost(t), _backendFnWithErr)
	require.Error(t, err)
	require.Nil(t, mod)
}

func TestNewWithOptionsError(t *testing.T) {
	mod, err := newAsyncModule(
		newTestHost(t),
		_memBackendFn,
		func(*Config) error { return errors.New("options error") },
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "options error")
	require.Nil(t, mod)
}

func createModule(t *testing.T, b BackendCreateFunc, options ...ModuleOption) Backend {
	moduleProvider := New(b, options...)
	assert.NotNil(t, moduleProvider)
	mod, err := moduleProvider.Create(newTestHost(t))
	assert.NotNil(t, mod)
	assert.NoError(t, err)
	return _globalBackend
}
