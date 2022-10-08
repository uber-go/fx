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

package exectest

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandSuccess(t *testing.T) {
	cmd := Command(t, func() {
		fmt.Println("hello world")
	})

	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(out))

	assert.True(t, cmd.ProcessState.Exited(), "must exit")
	assert.Zero(t, cmd.ProcessState.ExitCode(), "exit code")
}

func TestCommandNonZero(t *testing.T) {
	cmd := Command(t, func() {
		fmt.Fprintln(os.Stderr, "great sadness")
		os.Exit(1)
	})

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.Error(t, err, "command must fail")

	assert.Equal(t, "great sadness\n", stderr.String())
	assert.True(t, cmd.ProcessState.Exited(), "must exit")
	assert.Equal(t, 1, cmd.ProcessState.ExitCode(), "exit code")
}
