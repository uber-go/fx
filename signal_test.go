// Copyright (c) 2022 Uber Technologies, Inc.
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

package fx

import (
	"github.com/stretchr/testify/assert"
	"syscall"
	"testing"
)

func assertUnsentSignalError(
	t *testing.T,
	err error,
	expected unsentSignalError,
) {
	t.Helper()

	var actual unsentSignalError

	assert.Error(t, err)
	assert.ErrorContains(t, err, "channels are blocked")
	assert.ErrorAs(t, err, &actual, "is unsentSignalError")
	assert.Equal(t, expected, actual)
}

func TestSignal(t *testing.T) {
	t.Parallel()
	recv := new(signalReceivers)
	a := recv.done()
	b := recv.done()

	assert.NotNil(t, a)
	assert.NotNil(t, b)

	expected := Signal{
		OS: syscall.SIGUSR1,
	}

	err := recv.broadcast(expected)

	assert.NoError(t, err, "first broadcast should succeed")

	err = recv.broadcast(expected)

	assertUnsentSignalError(t, err, unsentSignalError{
		Signal:   expected,
		Channels: 2,
		Unsent:   2,
	})

	actual := <-a
	assert.Equal(t, expected.OS, actual)

	assert.Equal(t, expected.OS, <-recv.done(), "expect cached signal")
}
