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

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxclock"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// Reflection types for each of the supported hook function signatures. These
// are used in cases in which the Callable constraint matches a user-defined
// function type that cannot be converted to an underlying function type with
// a conventional conversion or type switch.
var (
	_reflFunc             = reflect.TypeOf(Func(nil))
	_reflErrorFunc        = reflect.TypeOf(ErrorFunc(nil))
	_reflContextFunc      = reflect.TypeOf(ContextFunc(nil))
	_reflContextErrorFunc = reflect.TypeOf(ContextErrorFunc(nil))
)

// Discrete function signatures that are allowed as part of a [Callable].
type (
	// A Func can be converted to a ContextErrorFunc.
	Func = func()
	// An ErrorFunc can be converted to a ContextErrorFunc.
	ErrorFunc = func() error
	// A ContextFunc can be converted to a ContextErrorFunc.
	ContextFunc = func(context.Context)
	// A ContextErrorFunc is used as a [Hook.OnStart] or [Hook.OnStop]
	// function.
	ContextErrorFunc = func(context.Context) error
)

// A Callable is a constraint that matches functions that are, or can be
// converted to, functions suitable for a Hook.
//
// Callable must be identical to [fx.HookFunc].
type Callable interface {
	~Func | ~ErrorFunc | ~ContextFunc | ~ContextErrorFunc
}

// Wrap wraps x into a ContextErrorFunc suitable for a Hook.
func Wrap[T Callable](x T) (ContextErrorFunc, string) {
	if x == nil {
		return nil, ""
	}

	switch fn := any(x).(type) {
	case Func:
		return func(context.Context) error {
			fn()
			return nil
		}, fxreflect.FuncName(x)
	case ErrorFunc:
		return func(context.Context) error {
			return fn()
		}, fxreflect.FuncName(x)
	case ContextFunc:
		return func(ctx context.Context) error {
			fn(ctx)
			return nil
		}, fxreflect.FuncName(x)
	case ContextErrorFunc:
		return fn, fxreflect.FuncName(x)
	}

	// Since (1) we're already using reflect in Fx, (2) we're not particularly
	// concerned with performance, and (3) unsafe would require discrete build
	// targets for appengine (etc), just use reflect to convert user-defined
	// function types to their underlying function types and then call Wrap
	// again with the converted value.
	reflVal := reflect.ValueOf(x)
	switch {
	case reflVal.CanConvert(_reflFunc):
		return Wrap(reflVal.Convert(_reflFunc).Interface().(Func))
	case reflVal.CanConvert(_reflErrorFunc):
		return Wrap(reflVal.Convert(_reflErrorFunc).Interface().(ErrorFunc))
	case reflVal.CanConvert(_reflContextFunc):
		return Wrap(reflVal.Convert(_reflContextFunc).Interface().(ContextFunc))
	default:
		// Is already convertible to ContextErrorFunc.
		return Wrap(reflVal.Convert(_reflContextErrorFunc).Interface().(ContextErrorFunc))
	}
}

// A Hook is a pair of start and stop callbacks, either of which can be nil,
// plus a string identifying the supplier of the hook.
type Hook struct {
	OnStart     func(context.Context) error
	OnStop      func(context.Context) error
	OnStartName string
	OnStopName  string

	callerFrame fxreflect.Frame
}

type appState int

const (
	stopped appState = iota
	starting
	incompleteStart
	started
	stopping
)

func (as appState) String() string {
	switch as {
	case stopped:
		return "stopped"
	case starting:
		return "starting"
	case incompleteStart:
		return "incompleteStart"
	case started:
		return "started"
	case stopping:
		return "stopping"
	default:
		return "invalidState"
	}
}

// Lifecycle coordinates application lifecycle hooks.
type Lifecycle struct {
	clock        fxclock.Clock
	logger       fxevent.Logger
	state        appState
	hooks        []Hook
	numStarted   int
	startRecords HookRecords
	stopRecords  HookRecords
	runningHook  Hook
	mu           sync.Mutex
}

// New constructs a new Lifecycle.
func New(logger fxevent.Logger, clock fxclock.Clock) *Lifecycle {
	return &Lifecycle{logger: logger, clock: clock}
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	// Save the caller's stack frame to report file/line number.
	if f := fxreflect.CallerStack(2, 0); len(f) > 0 {
		hook.callerFrame = f[0]
	}
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
func (l *Lifecycle) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("called OnStart with nil context")
	}

	l.mu.Lock()
	if l.state != stopped {
		defer l.mu.Unlock()
		return fmt.Errorf("attempted to start lifecycle when in state: %v", l.state)
	}
	l.numStarted = 0
	l.state = starting

	l.startRecords = make(HookRecords, 0, len(l.hooks))
	l.mu.Unlock()

	returnState := incompleteStart
	defer func() {
		l.mu.Lock()
		l.state = returnState
		l.mu.Unlock()
	}()

	for _, hook := range l.hooks {
		// if ctx has cancelled, bail out of the loop.
		if err := ctx.Err(); err != nil {
			return err
		}

		if hook.OnStart != nil {
			l.mu.Lock()
			l.runningHook = hook
			l.mu.Unlock()

			runtime, err := l.runStartHook(ctx, hook)
			if err != nil {
				return err
			}

			l.mu.Lock()
			l.startRecords = append(l.startRecords, HookRecord{
				CallerFrame: hook.callerFrame,
				Func:        hook.OnStart,
				Runtime:     runtime,
			})
			l.mu.Unlock()
		}
		l.numStarted++
	}

	returnState = started
	return nil
}

func (l *Lifecycle) runStartHook(ctx context.Context, hook Hook) (runtime time.Duration, err error) {
	funcName := hook.OnStartName
	if len(funcName) == 0 {
		funcName = fxreflect.FuncName(hook.OnStart)
	}

	l.logger.LogEvent(&fxevent.OnStartExecuting{
		CallerName:   hook.callerFrame.Function,
		FunctionName: funcName,
	})
	defer func() {
		l.logger.LogEvent(&fxevent.OnStartExecuted{
			CallerName:   hook.callerFrame.Function,
			FunctionName: funcName,
			Runtime:      runtime,
			Err:          err,
		})
	}()

	begin := l.clock.Now()
	err = hook.OnStart(ctx)
	return l.clock.Since(begin), err
}

// Stop runs any OnStop hooks whose OnStart counterpart succeeded. OnStop
// hooks run in reverse order.
func (l *Lifecycle) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("called OnStop with nil context")
	}

	l.mu.Lock()
	if l.state != started && l.state != incompleteStart && l.state != starting {
		defer l.mu.Unlock()
		return nil
	}
	l.state = stopping
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		l.state = stopped
		l.mu.Unlock()
	}()

	l.mu.Lock()
	l.stopRecords = make(HookRecords, 0, l.numStarted)
	// Take a snapshot of hook state to avoid races.
	allHooks := l.hooks[:]
	numStarted := l.numStarted
	l.mu.Unlock()

	// Run backward from last successful OnStart.
	var errs []error
	for ; numStarted > 0; numStarted-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		hook := allHooks[numStarted-1]
		if hook.OnStop == nil {
			continue
		}

		l.mu.Lock()
		l.runningHook = hook
		l.mu.Unlock()

		runtime, err := l.runStopHook(ctx, hook)
		if err != nil {
			// For best-effort cleanup, keep going after errors.
			errs = append(errs, err)
		}

		l.mu.Lock()
		l.stopRecords = append(l.stopRecords, HookRecord{
			CallerFrame: hook.callerFrame,
			Func:        hook.OnStop,
			Runtime:     runtime,
		})
		l.mu.Unlock()
	}

	return multierr.Combine(errs...)
}

func (l *Lifecycle) runStopHook(ctx context.Context, hook Hook) (runtime time.Duration, err error) {
	funcName := hook.OnStopName
	if len(funcName) == 0 {
		funcName = fxreflect.FuncName(hook.OnStop)
	}

	l.logger.LogEvent(&fxevent.OnStopExecuting{
		CallerName:   hook.callerFrame.Function,
		FunctionName: funcName,
	})
	defer func() {
		l.logger.LogEvent(&fxevent.OnStopExecuted{
			CallerName:   hook.callerFrame.Function,
			FunctionName: funcName,
			Runtime:      runtime,
			Err:          err,
		})
	}()

	begin := l.clock.Now()
	err = hook.OnStop(ctx)
	return l.clock.Since(begin), err
}

// RunningHookCaller returns the name of the hook that was running when a Start/Stop
// hook timed out.
func (l *Lifecycle) RunningHookCaller() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.runningHook.callerFrame.Function
}

// HookRecord keeps track of each Hook's execution time, the caller that appended the Hook, and function that ran as the Hook.
type HookRecord struct {
	CallerFrame fxreflect.Frame             // stack frame of the caller
	Func        func(context.Context) error // function that ran as sanitized name
	Runtime     time.Duration               // how long the hook ran
}

// HookRecords is a Stringer wrapper of HookRecord slice.
type HookRecords []HookRecord

func (rs HookRecords) Len() int {
	return len(rs)
}

func (rs HookRecords) Less(i, j int) bool {
	// Sort by runtime, greater ones at top.
	return rs[i].Runtime > rs[j].Runtime
}

func (rs HookRecords) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

// Used for logging startup errors.
func (rs HookRecords) String() string {
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "%s took %v from %s",
			fxreflect.FuncName(r.Func), r.Runtime, r.CallerFrame)
	}
	return b.String()
}

// Format implements fmt.Formatter to handle "%+v".
func (rs HookRecords) Format(w fmt.State, c rune) {
	if !w.Flag('+') {
		// Without %+v, fall back to String().
		io.WriteString(w, rs.String())
		return
	}

	for _, r := range rs {
		fmt.Fprintf(w, "\n%s took %v from:\n\t%+v",
			fxreflect.FuncName(r.Func),
			r.Runtime,
			r.CallerFrame)
	}
	fmt.Fprintf(w, "\n")
}
