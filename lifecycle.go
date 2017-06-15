// Copyright (c) 2017 Uber Technologies, Inc.
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
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// Lifecycle allows constructors to register callbacks that are executed on
// application start and stop.
type Lifecycle interface {
	Append(Hook)
}

// A Hook is a pair of start and stop callbacks, either of which can be nil.
// If a Hook's OnStart callback isn't executed (because a previous OnStart
// failure short-circuited application start), its OnStop callback won't be
// executed.
type Hook struct {
	OnStart func() error
	OnStop  func() error
	caller  string
}

func newLifecycle(logger fxlog.Logger) *lifecycle {
	if logger == nil {
		logger = fxlog.New()
	}
	return &lifecycle{
		logger: logger,
	}
}

type lifecycle struct {
	logger   fxlog.Logger
	hooks    []Hook
	position int
}

func (l *lifecycle) Append(hook Hook) {
	hook.caller = fxreflect.Caller()
	l.hooks = append(l.hooks, hook)
}

// start calls all OnStarts in order, halting on the first OnStart that fails
// and marking that position so that stop can rollback.
func (l *lifecycle) start() error {
	for i, hook := range l.hooks {
		if hook.OnStart != nil {
			l.logger.Printf("START\t\t%s()", hook.caller)
			if err := hook.OnStart(); err != nil {
				return err
			}
		}
		l.position = i
	}
	return nil
}

// stop calls all OnStops from the position of the last succeeding OnStart. If
// any OnStops fail, stop continues, doing a best-try cleanup. All errs are
// gathered and returned as a single error.
func (l *lifecycle) stop() error {
	if len(l.hooks) == 0 {
		return nil
	}
	var errs []error
	for i := l.position; i >= 0; i-- {
		if l.hooks[i].OnStop == nil {
			continue
		}
		l.logger.Printf("STOP\t\t%s()", l.hooks[i].caller)
		if err := l.hooks[i].OnStop(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// NewTestLifecycle creates a new test lifecycle.
func NewTestLifecycle() *TestLifecycle {
	return &TestLifecycle{
		newLifecycle(nil),
	}
}

// TestLifecycle makes testing funcs that rely on Lifecycle possible by
// exposing a Start and Stop func which can be called manually in unit tests.
type TestLifecycle struct {
	*lifecycle
}

var _ Lifecycle = (*TestLifecycle)(nil)

// Start the lifecycle
func (l *TestLifecycle) Start() error {
	return l.start()
}

// Stop the lifecycle
func (l *TestLifecycle) Stop() error {
	return l.stop()
}
