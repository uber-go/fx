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
)

func TestNopBackend(t *testing.T) {
	b := &NopBackend{}
	testBackendMethods(t, b)
	assert.NoError(t, b.Publish(nil, nil))
	assert.NoError(t, b.Stop())
}

func TestInMemBackend(t *testing.T) {
	b := NewInMemBackend(service.NopHost())
	errorCh := testBackendMethods(t, b)
	publishEncodedVal(t, b, errorCh)
	publishEncodedVal(t, b, errorCh)

	assert.NoError(t, b.Stop())
	assert.False(t, b.IsRunning())
}

func TestInMemBackendStartAfterStart(t *testing.T) {
	b := NewInMemBackend(service.NopHost())
	_ = b.Start(make(chan struct{}))
	errorCh := b.Start(make(chan struct{}))
	err := <-errorCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestInMemBackendStartAfterStop(t *testing.T) {
	b := NewInMemBackend(service.NopHost())
	_ = b.Stop()
	errorCh := b.Start(make(chan struct{}))
	err := <-errorCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "been stopped")
}

func TestInMemBackendStartTimeout(t *testing.T) {
	b := NewInMemBackend(service.NopHost())
	_ = b.Start(make(chan struct{}))
	defer b.Stop()
	time.Sleep(time.Millisecond)
}

func testBackendMethods(t *testing.T, b Backend) <-chan error {
	assert.NotEmpty(t, b.Name())
	assert.NotNil(t, b.Encoder())

	errorCh := b.Start(make(chan struct{}))
	assert.True(t, b.IsRunning())
	return errorCh
}

func publishEncodedVal(t *testing.T, b Backend, errorCh <-chan error) {
	bytes, err := b.Encoder().Marshal("Hello")
	assert.NoError(t, err)

	// Publish a non FnSignature value that results in error
	err = b.Publish(context.Background(), bytes)
	assert.NoError(t, err)
	if err, ok := <-errorCh; ok {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type mismatch")
	}
}
