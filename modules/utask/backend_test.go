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

import "testing"
import "github.com/stretchr/testify/assert"

func TestNopBackend(t *testing.T) {
	testBackendMethods(t, &NopBackend{})
}

func TestInMemBackend(t *testing.T) {
	testBackendMethods(t, &InMemBackend{})
}

func testBackendMethods(t *testing.T, backend Backend) {
	assert.NotNil(t, backend.Name())
	assert.NotNil(t, backend.Type())
	assert.NotNil(t, backend.Encoder())
	assert.NotNil(t, backend.Start(make(chan struct{})))
	assert.True(t, backend.IsRunning())
	// Testing that consume before publish does not throw an error. Not testing Consume after publish
	// since that is tested in execution and requires function setup etc.
	assert.NoError(t, backend.Consume())
	assert.NoError(t, backend.Publish([]byte{}, make(map[string]string)))
	assert.NoError(t, backend.Stop())
}
