// Copyright (c) 2020-2021 Uber Technologies, Inc.
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

//go:build windows
// +build windows

package fx_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"golang.org/x/sys/windows"
)

// Regression test for https://github.com/uber-go/fx/issues/781.
func TestWindowsCtrlCHandler(t *testing.T) {
	// This test operates by launching a separate process,
	// which we'll send a SIGINT to,
	// and verifying the output of the application.

	// Launch a separate process we will send SIGINT to.
	testExe, err := os.Executable()
	require.NoError(t, err, "determine test executable")
	cmd := exec.Command(testExe, "-test.run", "TestWindowsMinimalApp")
	cmd.Env = append(os.Environ(), "FX_TEST_FAKE=1")

	// On Windows, we need to use GenerateConsoleCtrlEvent
	// to SIGINT the child process.
	//
	// That API operates on Group ID granularity,
	// so we need to make sure our new child process
	// gets a new group ID rather than using the same ID
	// as the test we're running.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "create stdout")

	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "create stderr")

	require.NoError(t, cmd.Start())

	// Block until the child is ready by waiting for the "ready" text
	// printed to stderr.
	ready := make(chan struct{})
	go func() {
		defer close(ready)
		stderr.Read(make([]byte, 1024))
	}()
	<-ready

	require.NoError(t,
		windows.GenerateConsoleCtrlEvent(1, uint32(cmd.Process.Pid)),
		"SIGINT child process")

	// Drain stdout and stderr, and wait for the process to exit.
	output, err := ioutil.ReadAll(stdout)
	require.NoError(t, err)
	_, err = io.Copy(ioutil.Discard, stderr)
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())

	assert.Contains(t, string(output), "ONSTOP",
		"stdout should include ONSTOP")
}

func TestWindowsMinimalApp(t *testing.T) {
	// This is not a real test.
	// It defines the behavior of the fake application
	// that we spawn from TestWindowsCtrlCHandler.
	if os.Getenv("FX_TEST_FAKE") != "1" {
		return
	}

	// An Fx application that prints "ready" to stderr
	// once its start hooks have been invoked,
	// and "ONSTOP" to stdout when its stop hooks have been invoked.
	fx.New(
		fx.NopLogger,
		fx.Invoke(func(lifecycle fx.Lifecycle) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					fmt.Fprintln(os.Stderr, "ready")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Fprintln(os.Stdout, "ONSTOP")
					return nil
				},
			})
		}),
	).Run()
}
