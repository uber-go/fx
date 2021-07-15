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
func (*ProvideError) event()           {}
func (*Supply) event()                 {}
func (*Provide) event()                {}
func (*Invoke) event()                 {}
func (*InvokeError) event()            {}
func (*StartError) event()             {}
func (*StopSignal) event()             {}
func (*StopError) event()              {}
func (*Rollback) event()               {}
func (*RollbackError) event()          {}
func (*Running) event()                {}
func (*CustomLoggerError) event()      {}
func (*CustomLogger) event()           {}

// LifecycleHookExecuting is emitted before an OnStart hook is about to be executed.
type LifecycleHookExecuting struct {
	// FunctionName is the name of the hook being executed.
	FunctionName string
	// CallerName is the name of the caller that appended the hook.
	CallerName string
	// Method is the lifecycle hook method getting called.
	Method string
}

// LifecycleHookExecuted is emitted after an OnStart hook has been executed.
type LifecycleHookExecuted struct {
	FunctionName string
	CallerName   string
	Method       string
	Runtime      time.Duration
	Err          error
}

// ProvideError is emitted whenever there is an error applying options.
type ProvideError struct {
	Err error
}

// Supply is emitted whenever a Provide was called with a constructor provided
// by fx.Supply.
type Supply struct {
	TypeName string
}

// Provide is emitted when we add a constructor to the container.
type Provide struct {
	Constructor interface{}

	// OutputTypeNames is a list of names of types that are produced by
	// this constructor.
	OutputTypeNames []string
}

// Invoke is emitted whenever a function is invoked.
type Invoke struct {
	Function interface{}
}

// InvokeError is emitted when fx.Invoke has failed.
type InvokeError struct {
	Function   interface{}
	Err        error
	Stacktrace string
}

// StartError is emitted right before exiting after failing to start.
type StartError struct{ Err error }

// StopSignal is emitted whenever application receives a signal after
// starting the application.
type StopSignal struct{ Signal os.Signal }

// StopError is emitted whenever we fail to stop cleanly.
type StopError struct{ Err error }

// Rollback is emitted whenever a service fails to start.
type Rollback struct{ StartErr error }

// RollbackError is emitted whenever we fail to rollback cleanly after
// a start error.
type RollbackError struct{ Err error }

// Running is emitted whenever an application is started successfully.
type Running struct{}

// CustomLoggerError is emitted whenever a custom logger fails to construct.
type CustomLoggerError struct{ Err error }

// CustomLogger is emitted whenever a custom logger is set.
type CustomLogger struct {
	Function interface{}
}
