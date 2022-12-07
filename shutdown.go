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
	"context"
	"time"

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
// process.
type ShutdownOption interface {
	apply(*shutdowner)
}

type exitCodeOption int

var _ ShutdownOption = exitCodeOption(0)

func (code exitCodeOption) apply(s *shutdowner) {
	s.exitCode = int(code)
}

// ExitCode is a [ShutdownOption] that may be passed to the Shutdown method of the
// [Shutdowner] interface.
// The given integer exit code will be broadcasted to any receiver waiting
// on a [ShutdownSignal] from the [Wait] method.
func ExitCode(code int) ShutdownOption {
	return exitCodeOption(code)
}

type shutdownTimeoutOption time.Duration

func (to shutdownTimeoutOption) apply(s *shutdowner) {
	s.shutdownTimeout = time.Duration(to)
}

var _ ShutdownOption = shutdownTimeoutOption(0)

// ShutdownTimeout is a [ShutdownOption] that allows users to specify a timeout
// for a given call to Shutdown method of the [Shutdowner] interface. As the
// Shutdown method will block while waiting for a signal receiver relay
// goroutine to stop.
func ShutdownTimeout(timeout time.Duration) ShutdownOption {
	return shutdownTimeoutOption(timeout)
}

type shutdownErrorOption []error

func (errs shutdownErrorOption) apply(s *shutdowner) {
	s.app.err = multierr.Append(s.app.err, multierr.Combine(errs...))
}

var _ ShutdownOption = shutdownErrorOption([]error{})

// ShutdownError registers any number of errors with the application shutdown.
// If more than one error is given, the errors are combined into a
// single error. Similar to invocations, errors are applied in order.
//
// You can use these errors, for example, to decide what to do after the app shutdown.
//
//	customErr := errors.New("something went wrong")
//	app := fx.New(
//		...
//		fx.Provide(func(s fx.Shutdowner, a A) B {
//			s.Shutdown(fx.ShutdownError(customErr))
//	    }),
//	    ...
//	)
//	err := app.Start(context.Background())
//	if err != nil {
//		panic(err)
//	}
//	defer app.Stop(context.Background())
//
//	if err := app.Err(); errors.Is(err, customErr) {
//	   // custom logic here
//	}
func ShutdownError(errs ...error) ShutdownOption {
	return shutdownErrorOption(errs)
}

type shutdowner struct {
	app             *App
	exitCode        int
	shutdownTimeout time.Duration
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

	ctx := context.Background()

	if s.shutdownTimeout != time.Duration(0) {
		c, cancel := context.WithTimeout(
			context.Background(),
			s.shutdownTimeout,
		)
		defer cancel()
		ctx = c
	}

	defer s.app.receivers.Stop(ctx)

	return s.app.receivers.Broadcast(ShutdownSignal{
		Signal:   _sigTERM,
		ExitCode: s.exitCode,
	})
}

func (app *App) shutdowner() Shutdowner {
	return &shutdowner{app: app}
}
