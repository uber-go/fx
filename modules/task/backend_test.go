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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withBackendSetup(t *testing.T, backend Backend) func() {
	_globalBackend = backend
	require.NoError(t, _globalBackend.Start())
	return func() {
		assert.NoError(t, _globalBackend.Stop())
		_globalBackend = NopBackend{}
	}
}

func TestNopBackend(t *testing.T) {
	b := &NopBackend{}
	assert.NotNil(t, b.Encoder())
	assert.NoError(t, b.Start())
	assert.NoError(t, b.Enqueue(nil, nil))
	assert.NoError(t, b.Stop())
}

func TestInMemBackend(t *testing.T) {
	b := NewInMemBackend()
	defer withBackendSetup(t, b)()
	require.NotNil(t, b.Encoder())
	require.NoError(t, b.Start())
	require.NoError(t, b.ExecuteAsync())
	memB := b.(*inMemBackend)
	publishEncodedVal(t, memB)
	publishEncodedVal(t, memB)
}

func TestInMemBackendStartTimeout(t *testing.T) {
	b := NewInMemBackend()
	_ = b.Start()
	defer b.Stop()
	time.Sleep(time.Millisecond)
}

func publishEncodedVal(t *testing.T, b Backend) {
	bytes, err := b.Encoder().Marshal("Hello")
	assert.NoError(t, err)

	// Publish a non FnSignature value that results in error
	err = b.Enqueue(context.Background(), bytes)
	assert.NoError(t, err)
	errorCh := b.(*inMemBackend).errorCh
	require.NotNil(t, errorCh)
	if err, ok := <-errorCh; ok {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type mismatch")
	}
}
