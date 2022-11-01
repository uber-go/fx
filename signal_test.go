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
	expected *unsentSignalError,
) {
	t.Helper()

	actual := new(unsentSignalError)

	assert.ErrorContains(t, err, "channels are blocked")
	if assert.ErrorAs(t, err, &actual, "is unsentSignalError") {
		assert.Equal(t, expected, actual)
	}
}

func TestSignal(t *testing.T) {
	t.Parallel()
	recv := new(signalReceivers)
	a := recv.Done()
	b := recv.Done()


	expected := ShutdownSignal{
		OS: syscall.SIGTERM,
	}

	err := recv.Broadcast(expected)

	assert.NoError(t, err, "first broadcast should succeed")

	err = recv.Broadcast(expected)

	assertUnsentSignalError(t, err, &unsentSignalError{
		Signal:   expected,
		Channels: 2,
		Unsent:   2,
	})

	actual := <-a
	assert.Equal(t, expected.OS, actual)

	assert.Equal(t, expected.OS, <-recv.Done(), "expect cached signal")
}
