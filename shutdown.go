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
	"fmt"
	"os"

	"go.uber.org/multierr"
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

type shutdownCode int

func (c shutdownCode) apply(s *shutdowner) {
	s.exitCode = int(c)
}

// ShutdownCode implements a shutdown option that allows a user specify the
// os.Exit code that an application should exit with.
func ShutdownCode(code int) ShutdownOption {
	return shutdownCode(code)
}

type shutdowner struct {
	exitCode int
	app      *App
}

type ShutdownSignal struct {
	Signal   os.Signal
	ExitCode int
}

// Shutdown broadcasts a signal to all of the application's Done channels
// and begins the Stop process. Applications can be shut down only after they
// have finished starting up.
// In practice this means Shutdowner.Shutdown should not be called from an
// fx.Invoke, but from a fx.Lifecycle.OnStart hook.
func (s *shutdowner) Shutdown(opts ...ShutdownOption) error {
	for _, opt := range opts {
		opt.apply(s)
	}

	return s.app.broadcastSignal(_sigTERM, s.exitCode)
}

func (app *App) shutdowner() Shutdowner {
	return &shutdowner{app: app}
}

func (app *App) broadcastSignal(signal os.Signal, code int) error {
	return multierr.Combine(
		app.broadcastDoneSignal(signal),
		app.broadcastWaitSignal(signal, code),
	)
}

func (app *App) broadcastDoneSignal(signal os.Signal) error {
	app.donesMu.Lock()
	defer app.donesMu.Unlock()

	app.shutdownSig = signal

	var unsent int
	for _, done := range app.dones {
		select {
		case done <- signal:
		default:
			// shutdown called when done channel has already received a
			// termination signal that has not been cleared
			unsent++
		}
	}

	if unsent != 0 {
		return ErrOnUnsentSignal{
			Signal:   signal,
			Unsent:   unsent,
			Channels: len(app.dones),
		}
	}

	return nil
}

func (app *App) broadcastWaitSignal(signal os.Signal, code int) error {
	app.waitsMu.Lock()
	defer app.waitsMu.Unlock()

	app.shutdownSignal = &ShutdownSignal{
		Signal:   signal,
		ExitCode: code,
	}

	var unsent int
	for _, wait := range app.waits {
		select {
		case wait <- *app.shutdownSignal:
		default:
			// shutdown called when wait channel has already received a
			// termination signal that has not been cleared
			unsent++
		}
	}

	if unsent != 0 {
		return ErrOnUnsentSignal{
			Signal:   signal,
			Unsent:   unsent,
			Code:     code,
			Channels: len(app.waits),
		}
	}

	return nil
}

// ErrOnUnsentSignal ... TBD
type ErrOnUnsentSignal struct {
	Signal   os.Signal
	Unsent   int
	Code     int
	Channels int
}

func (err ErrOnUnsentSignal) Error() string {
	return fmt.Sprintf(
		"failed to send %v signal to %v out of %v channels",
		err.Signal,
		err.Unsent,
		err.Channels,
	)
}
