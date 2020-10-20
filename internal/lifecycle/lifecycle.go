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

package lifecycle

import (
	"context"
	"sort"

	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// Priority of a Hook in the Lifecycle queue
type Priority int

const (
	// PriorityLowest hooks are started last and stoppped first.
	// For example, hooks that start serving network traffic might wish to execute
	// only after all other application components are initialized.
	PriorityLowest Priority = -1000
	// PriorityNormal is the default priority level
	PriorityNormal = 0
	// PriorityHighest hooks are started first and stopped last.
	PriorityHighest = 1000
)

// A Hook is a pair of start and stop callbacks, either of which can be nil,
// plus a string identifying the supplier of the hook.
type Hook struct {
	OnStart  func(context.Context) error
	OnStop   func(context.Context) error
	Priority Priority
	caller   string
}

// Lifecycle coordinates application lifecycle hooks.
type Lifecycle struct {
	logger     *fxlog.Logger
	hooks      []Hook
	numStarted int
}

// New constructs a new Lifecycle.
func New(logger *fxlog.Logger) *Lifecycle {
	if logger == nil {
		logger = fxlog.New()
	}
	return &Lifecycle{logger: logger}
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	hook.caller = fxreflect.Caller()
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
func (l *Lifecycle) Start(ctx context.Context) error {
	// Highest priority to lowest priority
	sort.SliceStable(l.hooks, func(i, j int) bool {
		return l.hooks[i].Priority > l.hooks[j].Priority
	})
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
