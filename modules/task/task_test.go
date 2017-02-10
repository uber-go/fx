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
	_nopBackendFn = func(host service.Host) (Backend, error) { return _nopBackend, nil }
	_memBackend   = NewInMemBackend()
	_memBackendFn = func(host service.Host) (Backend, error) { return _memBackend, nil }
	_errBackendFn = func(host service.Host) (Backend, error) { return nil, errors.New("bknd err") }
	_mi           = service.ModuleCreateInfo{
		Host: service.NopHost(),
	}
)

func TestNewModule(t *testing.T) {
	b := createModule(t, _memBackendFn) // Singleton modules get saved
	require.Equal(t, _memBackend, b)
	b = createModule(t, _nopBackendFn) // Singleton returns nop even though mem backend is input
	require.Equal(t, _memBackend, b)
}

func TestNewModuleError(t *testing.T) {
	mod, err := newAsyncModule(_mi, _errBackendFn)
	require.Error(t, err)
	require.Nil(t, mod)
}

func TestMemBackendModuleWorkflowWithContext(t *testing.T) {
	b := createModule(t, _memBackendFn) // we will just get the singleton in mem backend here
	errChan := b.Start(make(chan struct{}))
	fn := func(ctx context.Context) error {
		fmt.Printf("Hello")
		return errors.New("hello error")
	}
	require.NoError(t, Register(fn))
	require.NoError(t, Enqueue(fn, context.Background()))
	require.Error(t, <-errChan)
}

func createModule(t *testing.T, b BackendCreateFunc) Backend {
	createFn := NewModule(b)
	assert.NotNil(t, createFn)
	mods, err := createFn(_mi)
	assert.NotNil(t, mods)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mods))
	return mods[0].(*AsyncModule).Backend
}
