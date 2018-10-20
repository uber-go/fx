// Copyright (c) 2018 Uber Technologies, Inc.
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

package lifecycle

import (
	"context"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// A Hook is a pair of start and stop callbacks, either of which can be nil,
// plus a string identifying the supplier of the hook.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
	caller  string
}

// Lifecycle coordinates application lifecycle hooks.
type Lifecycle struct {
	logger     *fxlog.Logger
	hooks      []Hook
	numStarted int
	stop       chan struct{}
}

// New constructs a new Lifecycle.
func New(logger *fxlog.Logger) *Lifecycle {
	if logger == nil {
		logger = fxlog.New()
	}
	return &Lifecycle{
		logger: logger,
		stop:   make(chan struct{}, 1),
	}
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	hook.caller = fxreflect.Caller()
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
func (l *Lifecycle) Start(ctx context.Context) error {
	for _, hook := range l.hooks {
		if hook.OnStart != nil {
			l.logger.Printf("START\t\t%s()", hook.caller)
			if err := hook.OnStart(ctx); err != nil {
				return err
			}
		}
		l.numStarted++
	}
	return nil
}

// Stop runs any OnStop hooks whose OnStart counterpart succeeded. OnStop
// hooks run in reverse order.
func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error
	// Run backward from last successful OnStart.
	for ; l.numStarted > 0; l.numStarted-- {
		hook := l.hooks[l.numStarted-1]
		if hook.OnStop == nil {
			continue
		}
		l.logger.Printf("STOP\t\t%s()", hook.caller)
		if err := hook.OnStop(ctx); err != nil {
			// For best-effort cleanup, keep going after errors.
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// Run ...
func (l *Lifecycle) Run(startTimeout time.Duration, stopTimeout time.Duration) (err error) {
	return l.run(startTimeout, stopTimeout, l.Done())
}

// Shutdown stops
func (l *Lifecycle) Shutdown() {
	l.stop <- struct{}{}
}

// Done returns a channel of signals to block on after starting the
// application. Applications listen for the SIGINT and SIGTERM signals; during
// development, users can send the application SIGTERM by pressing Ctrl-C in
// the same terminal as the running process.
func (l *Lifecycle) Done() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}

// Run ...
func (l *Lifecycle) run(startTimeout time.Duration, stopTimeout time.Duration, done <-chan os.Signal) (err error) {
	startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)
	defer startCancel()

	if err = withTimeout(startCtx, l.Start); err != nil {
		return err
	}

	l.logger.Printf("RUNNING")

	select {
	case s := <-done:
		l.logger.PrintSignal(s)
	case <-l.stop:
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()

	if err = withTimeout(stopCtx, l.Stop); err != nil {
		return err
	}

	return nil
}

func withTimeout(ctx context.Context, f func(context.Context) error) error {
	c := make(chan error, 1)
	go func() { c <- f(ctx) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}
