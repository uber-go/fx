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
	"errors"
	"testing"

	"golang.org/x/net/context"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_nopBackend   = &NopBackend{}
	_nopBackendFn = func(service.Host) (Backend, error) { return _nopBackend, nil }
	_memBackend   *inMemBackend
	_memBackendFn = func(service.Host) (Backend, error) { return _memBackend, nil }
	_errBackendFn = func(service.Host) (Backend, error) { return nil, errors.New("bknd err") }
)

func init() {
	host, _ := service.NewScopedHost(service.NopHost(), "task", "hello")
	_memBackend = NewInMemBackend(host).(*inMemBackend)
}

func TestNew(t *testing.T) {
	b := createModule(t, _memBackendFn) // Singleton modules get saved
	require.Equal(t, _memBackend, b.(*managedBackend).Backend)
	b = createModule(t, _nopBackendFn) // Singleton returns nop even though mem backend is input
	require.Equal(t, _memBackend, b.(*managedBackend).Backend)
}

func TestMemBackendModuleWorkflowWithContext(t *testing.T) {
	mod, err := newAsyncModule(newTestHost(t), _memBackendFn, DisableExecution()) // we will just get the singleton in mem backend here
	require.NoError(t, err)
	require.NotNil(t, mod)
	b := mod.(Backend)
	require.True(t, b.(*managedBackend).config.DisableExecution)
	require.NoError(t, b.Start())
	fn := func(ctx context.Context) error {
		return errors.New("hello error")
	}
	errorCh := b.ExecuteAsync()
	require.NotNil(t, errorCh)
	require.NoError(t, Register(fn))
	require.NoError(t, Enqueue(fn, context.Background()))
	require.Error(t, <-errorCh)
}

func TestNewError(t *testing.T) {
	mod, err := newAsyncModule(newTestHost(t), _errBackendFn)
	require.Error(t, err)
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
