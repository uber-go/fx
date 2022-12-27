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

package fx

import (
	"context"

	"go.uber.org/fx/internal/lifecycle"
)

// A HookFunc is a function that can be used as a [Hook].
type HookFunc interface {
	~func() | ~func() error | ~func(context.Context) | ~func(context.Context) error
}

// Lifecycle allows constructors to register callbacks that are executed on
// application start and stop. See the documentation for App for details on Fx
// applications' initialization, startup, and shutdown logic.
type Lifecycle interface {
	Append(Hook)
}

// A Hook is a pair of start and stop callbacks, either of which can be nil.
// If a Hook's OnStart callback isn't executed (because a previous OnStart
// failure short-circuited application startup), its OnStop callback won't be
// executed.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error

	onStartName string
	onStopName  string
}

// StartHook returns a new Hook with start as its [Hook.OnStart] function,
// wrapping its signature as needed. For example, given the following function:
//
//	func myhook() {
//	  fmt.Println("hook called")
//	}
//
// then calling:
//
//	lifecycle.Append(StartHook(myfunc))
//
// is functionally equivalent to calling:
//
//	lifecycle.Append(fx.Hook{
//	  OnStart: func(context.Context) error {
//	    myfunc()
//	    return nil
//	  },
//	})
//
// The same is true for all functions that satisfy the HookFunc constraint.
// Note that any context.Context parameter or error return will be propagated
// as expected. If propagation is not intended, users should instead provide a
// closure that discards the undesired value(s), or construct a Hook directly.
func StartHook[T HookFunc](start T) Hook {
	onstart, startname := lifecycle.Wrap(start)

	return Hook{
		OnStart:     onstart,
		onStartName: startname,
	}
}

// StopHook returns a new Hook with stop as its [Hook.OnStop] function,
// wrapping its signature as needed. For example, given the following function:
//
//	func myhook() {
//	  fmt.Println("hook called")
//	}
//
// then calling:
//
//	lifecycle.Append(StopHook(myfunc))
//
// is functionally equivalent to calling:
//
//	lifecycle.Append(fx.Hook{
//	  OnStop: func(context.Context) error {
//	    myfunc()
//	    return nil
//	  },
//	})
//
// The same is true for all functions that satisfy the HookFunc constraint.
// Note that any context.Context parameter or error return will be propagated
// as expected. If propagation is not intended, users should instead provide a
// closure that discards the undesired value(s), or construct a Hook directly.
func StopHook[T HookFunc](stop T) Hook {
	onstop, stopname := lifecycle.Wrap(stop)

	return Hook{
		OnStop:     onstop,
		onStopName: stopname,
	}
}

// StartStopHook returns a new Hook with start as its [Hook.OnStart] function
// and stop as its [Hook.OnStop] function, independently wrapping the signature
// of each as needed.
func StartStopHook[T1 HookFunc, T2 HookFunc](start T1, stop T2) Hook {
	var (
		onstart, startname = lifecycle.Wrap(start)
		onstop, stopname   = lifecycle.Wrap(stop)
	)

	return Hook{
		OnStart:     onstart,
		OnStop:      onstop,
		onStartName: startname,
		onStopName:  stopname,
	}
}

type lifecycleWrapper struct {
	*lifecycle.Lifecycle
}

func (l *lifecycleWrapper) Append(h Hook) {
	l.Lifecycle.Append(lifecycle.Hook{
		OnStart:     h.OnStart,
		OnStop:      h.OnStop,
		OnStartName: h.onStartName,
		OnStopName:  h.onStopName,
	})
}
