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
	"io"
	"os"
	"os/exec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/docs/internal/test"
)

// StartWithOutput starts the given command,
// and returns an io.Reader that reads from both,
// its stdout and stderr.
//
// At test end, the system will ensure that
// the command finished running.
func StartWithOutput(t test.T, cmd *exec.Cmd) io.Reader {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err, "create pipe")

	cmd.Stdout = w
	cmd.Stderr = w
	require.NoError(t, cmd.Start(), "start command")
	// Close the output writer because this process won't write to it
	// anymore. Only the spawned process will.
	assert.NoError(t, w.Close(), "close output writer")
	t.Cleanup(func() {
		_, err := cmd.Process.Wait()
		assert.NoError(t, err, "wait for end")
		assert.NoError(t, r.Close(), "close output reader")
	})
	return r
}
