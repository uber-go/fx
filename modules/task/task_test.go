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
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
)

var (
	_nopBackend = &NopBackend{}
	_memBackend = NewInMemBackend()
	_mi         = service.ModuleCreateInfo{
		Host: service.NopHost(),
	}
)

func TestNewModule(t *testing.T) {
	createModule(t, _nopBackend) // Singleton modules get saved
	createModule(t, _memBackend) // Even though backend causes error, module saved earlier will return
}

func createModule(t *testing.T, b Backend) {
	createFn := NewModule(b)
	assert.NotNil(t, createFn)
	mods, err := createFn(_mi)
	assert.NotNil(t, mods)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mods))
	assert.Equal(t, _nopBackend, mods[0].(*AsyncModule).Backend)
}
