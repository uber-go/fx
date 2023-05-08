// Copyright (c) 2023 Uber Technologies, Inc.
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

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/internal/testutil"
)

// Hijacks the test binary so that the test can run main() as a subprocess
// instead of trying to compile the program and run it directly.
func TestMain(m *testing.M) {
	// If the test binary is named "app", then we're running as a subprocess.
	// Otherwise, run the tests.
	switch filepath.Base(os.Args[0]) {
	case "app":
		main()
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

// Verifies that an Fx program running with Run
// exits with the exit code passed to Shutdowner.
//
// Regression test for https://github.com/uber-go/fx/issues/1074.
func TestShutdownExitCode(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)

	out := testutil.WriteSyncer{T: t}

	// Run the test binary with the name 'app' so that it runs main().
	cmd := exec.Command(exe)
	cmd.Args[0] = "app"
	cmd.Stdout = &out
	cmd.Stderr = &out

	// The program should exit with code 20.
	err = cmd.Run()
	require.Error(t, err)

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)

	assert.Equal(t, 20, exitErr.ExitCode())
}
