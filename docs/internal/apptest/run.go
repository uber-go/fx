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

package apptest

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/docs/internal/exectest"
	"go.uber.org/fx/docs/internal/test"
)

// StartOption is an option for the Start function.
type StartOption interface{ apply(*startOptions) }

type startOptions struct {
	IsRunning func(string) bool
	Timeout   time.Duration
}

type isRunningOption func(string) bool

// IsRunning customizes how Start determines
// whether a log statement represents a running application.
//
// Defaults to DefaultIsRunning.
func IsRunning(f func(string) bool) StartOption {
	return isRunningOption(f)
}

func (o isRunningOption) apply(opts *startOptions) {
	opts.IsRunning = o
}

// DefaultIsRunning looks for lines in Fx's log output,
// which match either the ConsoleLogger or the ZapLogger's
// output,
// and represent that the application is running
// and ready to receive requests.
func DefaultIsRunning(line string) bool {
	// ConsoleLogger
	if strings.Contains(line, "[Fx] RUNNING") {
		return true
	}

	// ZapLogger
	var log struct {
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(line), &log); err == nil {
		return log.Msg == "started"
	}

	return false
}

type timeoutOption time.Duration

// Timeout specifies the duration after which we'll
// stop waiting for the application to start up.
//
// Defaults to 5s.
func Timeout(t time.Duration) StartOption {
	return timeoutOption(t)
}

func (o timeoutOption) apply(opts *startOptions) {
	opts.Timeout = time.Duration(o)
}

// Start starts the Fx application that main represents,
// and blocks until isRunning(line) reports true for a line of output.
// When the test exits, this signals for the application to stop,
// and waits for it to stop.
func Start(t test.T, main func(), options ...StartOption) {
	t.Helper()

	opts := startOptions{
		Timeout:   5 * time.Second,
		IsRunning: DefaultIsRunning,
	}
	for _, o := range options {
		o.apply(&opts)
	}

	cmd := exectest.Command(t, main)

	r := exectest.StartWithOutput(t, cmd)
	done := make(chan struct{})
	unblock := make(chan struct{})
	go func() {
		defer close(done)

		var found bool
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			line := scan.Text()
			t.Logf("%s", line)
			if !found && opts.IsRunning(line) {
				found = true
				close(unblock)
			}
		}
		assert.NoError(t, scan.Err(), "scan error")
	}()
	t.Cleanup(func() {
		assert.NoError(t, cmd.Process.Signal(os.Interrupt), "send SIGINT")
		<-done
	})

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	select {
	case <-unblock:
		return

	case <-done:
		// If the application exited without printing the Running
		// message, something went wrong.
		// Fail the test.
		t.Errorf("application exited unexpectedly")
		t.FailNow()

	case <-ctx.Done():
		// Application did not start within the specified timeout.
		t.Errorf("application did not start in %v", opts.Timeout)
		t.FailNow()
	}
}
