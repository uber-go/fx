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
	"bytes"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestCtrlCHandler(t *testing.T) {
	// Launch a separate process we will send SIGINT to.
	bin, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(bin)

	// buffers used to capture the output of the child process.
	so, _ := cmd.StdoutPipe()
	se, _ := cmd.StderrPipe()

	cmd.Env = []string{"VerifySignalHandler=1"}
	// CREATE_NEW_PROCESS_GROUP is required to send SIGINT to
	// the child process.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
	err = cmd.Start()
	require.NoError(t, err)
	childPid := cmd.Process.Pid

	c := make(chan struct{}, 1)

	go func() {
		se.Read(make([]byte, 1024))
		c <- struct{}{}
	}()

	// block until child proc is ready.
	<-c

	// Send signal to child proc.
	err = windows.GenerateConsoleCtrlEvent(1, uint32(childPid))
	require.NoError(t, err)

	// Drain out stdout/stderr before waiting.
	buf := new(bytes.Buffer)
	buf.ReadFrom(se)
	buf.ReadFrom(so)

	// Wait till child proc finishes
	err = cmd.Wait()

	// stdout should have ONSTOP printed on it from OnStop handler.
	assert.Contains(t, buf.String(), "ONSTOP")
	assert.NoError(t, err)
}
