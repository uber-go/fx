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
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/multierr"

	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
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
	logger     fxlog.Logger
	hooks      []Hook
	numStarted int
}

// New constructs a new Lifecycle.
func New(logger fxlog.Logger) *Lifecycle {
	return &Lifecycle{logger: logger}
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	hook.caller = fxreflect.Caller()
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
func (l *Lifecycle) Start(ctx context.Context, caller chan string, recorder chan HookRecord) error {
	defer close(caller)
	defer close(recorder)
	for _, hook := range l.hooks {
		if hook.OnStart != nil {
			fxlog.Info("starting", fxlog.Field{
				Key:   "caller",
				Value: hook.caller,
			}).Write(l.logger)
			caller <- hook.caller
			begin := time.Now()
			if err := hook.OnStart(ctx); err != nil {
				return err
			}
			recorder <- HookRecord{
				Runtime: time.Now().Sub(begin),
				Caller:  hook.caller,
				Func:    hook.OnStart,
			}
		}
		l.numStarted++
	}
	return nil
}

// Stop runs any OnStop hooks whose OnStart counterpart succeeded. OnStop
// hooks run in reverse order.
func (l *Lifecycle) Stop(ctx context.Context, c chan string, r chan HookRecord) error {
	defer close(c)
	defer close(r)
	var errs []error
	// Run backward from last successful OnStart.
	for ; l.numStarted > 0; l.numStarted-- {
		hook := l.hooks[l.numStarted-1]
		if hook.OnStop == nil {
			continue
		}
		fxlog.Info("stopping", fxlog.Field{
			Key:   "caller",
			Value: hook.caller,
		}).Write(l.logger)
		c <- hook.caller
		begin := time.Now()
		if err := hook.OnStop(ctx); err != nil {
			// For best-effort cleanup, keep going after errors.
			errs = append(errs, err)
		}
		r <- HookRecord{
			Runtime: time.Now().Sub(begin),
			Caller:  hook.caller,
			Func:    hook.OnStop,
		}
	}
	return multierr.Combine(errs...)
}

// HookRecord keeps track of each Hook's execution time, the caller that appended the Hook, and function that ran as the Hook.
type HookRecord struct {
	Runtime time.Duration               // How long the hook ran
	Caller  string                      // caller that appended this hook
	Func    func(context.Context) error // function that ran as sanitized name
}

// HookRecords is a Stringer wrapper of HookRecord slice.
type HookRecords []HookRecord

// Used for logging startup errors.
func (r HookRecords) String() string {
	var b strings.Builder
	sort.Slice(r, func(i, j int) bool { return r[i].Runtime < r[j].Runtime })
	for _, r := range r {
		b.WriteString(fmt.Sprintf("Hook: %s took %d ms to run. (Caller: %s)\n", fxreflect.FuncName(r.Func), r.Runtime.Milliseconds(), r.Caller))
	}
	return b.String()
}
