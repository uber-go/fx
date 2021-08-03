// Copyright (c) 2021 Uber Technologies, Inc.
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

package fxevent

import (
	"os"
	"time"
)

// Event defines an event emitted by fx.
type Event interface {
	event() // Only fxlog can implement this interface.
}

// Passing events by type to make Event hashable in the future.
func (*LifecycleHookExecuting) event() {}
func (*LifecycleHookExecuted) event()  {}
func (*Supplied) event()               {}
func (*Provide) event()                {}
func (*Invoking) event()               {}
func (*Invoked) event()                {}
func (*Stop) event()                   {}
func (*Rollback) event()               {}
func (*Started) event()                {}
func (*LoggerInitialized) event()      {}

// LifecycleHookExecuting is emitted before an OnStart or OnStop hook is
// executed.
type LifecycleHookExecuting struct {
	// FunctionName is the name of the function that will be executed.
	FunctionName string

	// CallerName is the name of the function that scheduled the hook for
	// execution.
	CallerName string

	// Method specifies the kind of hook we're executing. This is one of
	// "OnStart" and "OnStop".
	Method string
}

// LifecycleHookExecuted is emitted after an OnStart or OnStop hook has been
// executed.
type LifecycleHookExecuted struct {
	// FunctionName is the name of the function that was executed.
	FunctionName string

	// CallerName is the name of the function that scheduled the hook for
	// execution.
	CallerName string

	// Method specifies the kind of the hook. This is one of "OnStart" and
	// "OnStop".
	Method string

	// Runtime specifies how long it took to run this hook.
	Runtime time.Duration

	// Err is non-nil if the hook failed to execute.
	Err error
}

// Supplied is emitted whenever a Provide was called with a constructor provided
// by fx.Supply.
type Supplied struct {
	TypeName string
}

// Provide is emitted when we add a constructor to the container.
type Provide struct {
	Constructor interface{}

	// OutputTypeNames is a list of names of types that are produced by
	// this constructor.
	OutputTypeNames []string

	// Err is emitted if there was an error applying options.
	Err error
}

// Invoking is emitted whenever a function is being invoked.
type Invoking struct {
	Function interface{}
}

// Invoked is emitted whenever a function being invoked errored.
type Invoked struct {
	Function   interface{}
	Err        error
	Stacktrace string
}

// Started is emitted whenever an application is started successfully and/or
// it errored.
type Started struct{ Err error }

// Stop is emitted whenever application receives a signal after
// starting the application with an optional error.
type Stop struct {
	Signal os.Signal
	Err    error
}

// Rollback is emitted whenever a service fails to start with initial startup
// error and then optional error if rollback itself fails.
type Rollback struct {
	StartErr error
	Err      error
}

// LoggerInitialized is emitted whenever a custom logger is set or produces an error.
type LoggerInitialized struct {
	Constructor interface{}
	Err         error
}
