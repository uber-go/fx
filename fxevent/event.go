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
func (*Provided) event()               {}
func (*Invoking) event()               {}
func (*Invoked) event()                {}
func (*Stopping) event()               {}
func (*Stopped) event()                {}
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

// Supplied is emitted after a value is added with fx.Supply.
type Supplied struct {
	// Name of the type of value that was added.
	TypeName string

	// Err is non-nil if we failed to supply the value.
	Err error
}

// Provided is emitted when a constructor is provided to Fx.
type Provided struct {
	// ConstructorName is the name of the constructor that was provided to
	// Fx.
	ConstructorName string

	// OutputTypeNames is a list of names of types that are produced by
	// this constructor.
	OutputTypeNames []string

	// Err is non-nil if we failed to provide this constructor.
	Err error
}

// Invoking is emitted before we invoke a function specified with fx.Invoke.
type Invoking struct {
	// FunctionName is the name of the function that will be invoked.
	FunctionName string
}

// Invoked is emitted after we invoke a function specified with fx.Invoke,
// whether it succeded or failed.
type Invoked struct {
	// Functionname is the name of the function that was invoked.
	FunctionName string

	// Err is non-nil if the function failed to execute.
	Err error

	// Trace records information about where the fx.Invoke call was made.
	// Note that this is NOT a stack trace of the error itself.
	Trace string
}

// Started is emitted when an application is started successfully and/or it
// errored.
type Started struct {
	// Err is non-nil if the application failed to start successfully.
	Err error
}

// Stopping is emitted when the application receives a signal to shut down
// after starting. This may happen with fx.Shutdowner or by sending a signal to
// the application on the command line.
type Stopping struct {
	// Signal is the signal that caused this shutdown.
	Signal os.Signal
}

// Stopped is emitted when the application has finished shutting down, whether
// successfully or not.
type Stopped struct {
	// Err is non-nil if errors were encountered during shutdown.
	Err error
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
