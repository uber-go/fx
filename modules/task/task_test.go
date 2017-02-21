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
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_nopBackend   = &NopBackend{}
	_nopBackendFn = func(moduleInfo service.ModuleInfo) (Backend, error) { return _nopBackend, nil }
	_memBackend   *inMemBackend
	_memBackendFn = func(moduleInfo service.ModuleInfo) (Backend, error) { return _memBackend, nil }
	_errBackendFn = func(moduleInfo service.ModuleInfo) (Backend, error) { return nil, errors.New("bknd err") }
)

func init() {
	moduleInfo, _ := service.NewModuleInfo(service.NopHost(), "hello")
	_memBackend = NewInMemBackend(moduleInfo).(*inMemBackend)
}

func TestNewModule(t *testing.T) {
	b := createModule(t, _memBackendFn) // Singleton modules get saved
	require.Equal(t, _memBackend, b)
	b = createModule(t, _nopBackendFn) // Singleton returns nop even though mem backend is input
	require.Equal(t, _memBackend, b)
}

func TestNewModuleError(t *testing.T) {
	mod, err := newAsyncModule(newTestModuleInfo(t), _errBackendFn)
	require.Error(t, err)
	require.Nil(t, mod)
}

func TestMemBackendModuleWorkflowWithContext(t *testing.T) {
	b := createModule(t, _memBackendFn) // we will just get the singleton in mem backend here
	require.NoError(t, b.Start())
	fn := func(ctx context.Context) error {
		fmt.Printf("Hello")
		return errors.New("hello error")
	}
	require.NoError(t, Register(fn))
	require.NoError(t, Enqueue(fn, context.Background()))
	require.Error(t, <-_memBackend.ErrorCh())
}

func createModule(t *testing.T, b BackendCreateFunc) Backend {
	createFn := NewModule(b)
	assert.NotNil(t, createFn)
	mod, err := createFn(newTestModuleInfo(t))
	assert.NotNil(t, mod)
	assert.NoError(t, err)
	return _globalBackend
}
