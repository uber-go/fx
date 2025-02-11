// Copyright (c) 2019 Uber Technologies, Inc.
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
	"time"
)

// Shutdowner provides a method that can manually trigger the shutdown of the
// application by sending a signal to all open Done channels. Shutdowner works
// on applications using Run as well as Start, Done, and Stop. The Shutdowner is
// provided to all Fx applications.
type Shutdowner interface {
	Shutdown(...ShutdownOption) error
}

// ShutdownOption provides a way to configure properties of the shutdown
// process. Currently, no options have been implemented.
type ShutdownOption interface {
	apply(*shutdowner)
}

type exitCodeOption int

func (code exitCodeOption) apply(s *shutdowner) {
	s.exitCode = int(code)
}

var _ ShutdownOption = exitCodeOption(0)

// ExitCode is a [ShutdownOption] that may be passed to the Shutdown method of the
// [Shutdowner] interface.
// The given integer exit code will be broadcasted to any receiver waiting
// on a [ShutdownSignal] from the [Wait] method.
func ExitCode(code int) ShutdownOption {
	return exitCodeOption(code)
}

type shutdownTimeoutOption time.Duration

func (shutdownTimeoutOption) apply(*shutdowner) {}

var _ ShutdownOption = shutdownTimeoutOption(0)

// ShutdownTimeout is a [ShutdownOption] that allows users to specify a timeout
// for a given call to Shutdown method of the [Shutdowner] interface. As the
// Shutdown method will block while waiting for a signal receiver relay
// goroutine to stop.
//
// Deprecated: This option has no effect. Shutdown is not a blocking operation.
func ShutdownTimeout(timeout time.Duration) ShutdownOption {
	return shutdownTimeoutOption(timeout)
}

type shutdowner struct {
	app      *App
	exitCode int
}

// Shutdown broadcasts a signal to all of the application's Done channels
// and begins the Stop process. Applications can be shut down only after they
// have finished starting up.
func (s *shutdowner) Shutdown(opts ...ShutdownOption) error {
	for _, opt := range opts {
		opt.apply(s)
	}

	return s.app.receivers.b.Broadcast(ShutdownSignal{
		Signal:   _sigTERM,
		ExitCode: s.exitCode,
	})
}

func (app *App) shutdowner() Shutdowner {
	return &shutdowner{app: app}
}
