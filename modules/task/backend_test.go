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
	"testing"
	"time"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withBackendSetup(t *testing.T, backend Backend) func() {
	_globalBackend = backend
	_ = _globalBackend.Start()
	return func() {
		assert.NoError(t, _globalBackend.Stop())
		_globalBackend = NopBackend{}
	}
}

func TestNopBackend(t *testing.T) {
	b := &NopBackend{}
	assert.NotNil(t, b.Encoder())
	assert.NoError(t, b.Start())
	assert.NoError(t, b.Publish(nil, nil))
	assert.NoError(t, b.Stop())
}

func TestInMemBackend(t *testing.T) {
	b := NewInMemBackend(newTestHost(t)).(*inMemBackend)
	defer withBackendSetup(t, b)()
	assert.NotNil(t, b.Encoder())
	assert.NoError(t, b.Start())
	publishEncodedVal(t, b)
	publishEncodedVal(t, b)
}

func TestInMemBackendStartTimeout(t *testing.T) {
	b := NewInMemBackend(newTestHost(t))
	_ = b.Start()
	defer b.Stop()
	time.Sleep(time.Millisecond)
}

func publishEncodedVal(t *testing.T, b *inMemBackend) {
	bytes, err := b.Encoder().Marshal("Hello")
	assert.NoError(t, err)

	// Publish a non FnSignature value that results in error
	err = b.Publish(context.Background(), bytes)
	assert.NoError(t, err)
	if err, ok := <-b.ErrorCh(); ok {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type mismatch")
	}
}

func newTestHost(t *testing.T) service.Host {
	mi, err := service.NewScopedHost(service.NopHost(), "task", "hello")
	require.NoError(t, err)
	return mi
}
