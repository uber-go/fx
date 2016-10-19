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

package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubObserver_OnInit(t *testing.T) {
	o := observerStub()
	require.NoError(t, o.OnInit(&serviceHost{}))

	assert.True(t, o.init)
}

func TestStubObserver_OnStateChange(t *testing.T) {
	o := observerStub()
	o.OnStateChange(Uninitialized, Initialized)

	assert.Equal(t, Initialized, o.state)
}

func TestStubObserver_OnShutdown(t *testing.T) {
	o := observerStub()
	o.OnShutdown(ServiceExit{})

	assert.True(t, o.shutdown)
}

func TestStubObserver_OnCriticalError(t *testing.T) {
	o := observerStub()
	assert.False(t, o.OnCriticalError(errors.New("dying")))
}

func observerStub() *StubObserver {
	return &StubObserver{}
}
