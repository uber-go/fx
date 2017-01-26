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

import "testing"
import "github.com/stretchr/testify/assert"

var backends = []Backend{
	&NopBackend{}, NewInMemBackend(),
}

func TestBackends(t *testing.T) {
	for _, b := range backends {
		testBackendMethods(t, b)
	}
}

func TestInMemBackendConsumeAfterClose(t *testing.T) {
	b := NewInMemBackend()
	assert.NoError(t, b.Stop())
	err := b.Consume()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel has been closed")
}

func testBackendMethods(t *testing.T, b Backend) {
	assert.NotEmpty(t, b.Name())
	assert.NotEmpty(t, b.Type())
	assert.NotNil(t, b.Encoder())
	assert.NotNil(t, b.Start(make(chan struct{})))
	assert.True(t, b.IsRunning())
	// Testing that consume before publish does not throw an error. Not testing Consume after publish
	// since that is tested in execution and requires function setup etc.
	assert.NoError(t, b.Consume())
	assert.NoError(t, b.Publish([]byte{}, make(map[string]string)))
	assert.NoError(t, b.Stop())
}
