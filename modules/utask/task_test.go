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
	"testing"

	"go.uber.org/fx/service"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	_nopBackend = &NopBackend{}
)

func TestModule(t *testing.T) {
	backendRegisterFn := func(service.Host, Config) (Backend, error) {
		return _nopBackend, nil
	}
	mods, err := createModules(t, backendRegisterFn)
	assert.NoError(t, err)
	assert.NotNil(t, mods)
	assert.Equal(t, 1, len(mods))
	assert.Equal(t, _nopBackend, mods[0].(*AsyncModule).Backend)
}

func TestModuleError(t *testing.T) {
	backendRegisterFn := func(service.Host, Config) (Backend, error) {
		return nil, errors.New("backend register error")
	}
	mods, err := createModules(t, backendRegisterFn)
	assert.Nil(t, mods)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backend register error")
}

func createModules(t *testing.T, fn backendRegisterFn) ([]service.Module, error) {
	createFn := NewModule()
	assert.NotNil(t, createFn)

	mi := service.ModuleCreateInfo{
		Host: service.NopHost(),
	}
	RegisterAsyncBackend(fn)
	return createFn(mi)
}
