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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// richError prints a different output when formatted with %+v vs %v.
type richError struct{}

func (e *richError) Error() string { return "plain error" }

func (e *richError) Format(w fmt.State, c rune) {
	if w.Flag('+') && c == 'v' {
		// Format differently for %+v.
		io.WriteString(w, "rich error")
	} else {
		io.WriteString(w, e.Error())
	}
}

func TestConsoleLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give Event
		want string
	}{
		{
			name: "OnStart executing",
			give: &OnStartExecuting{
				FunctionName: "hook.onStart",
				CallerName:   "bytes.NewBuffer",
			},
			want: "[Fx] HOOK OnStart		hook.onStart executing (caller: bytes.NewBuffer)\n",
		},
		{
			name: "OnStopExecuting",
			give: &OnStopExecuting{
				FunctionName: "hook.onStop1",
				CallerName:   "bytes.NewBuffer",
			},
			want: "[Fx] HOOK OnStop		hook.onStop1 executing (caller: bytes.NewBuffer)\n",
		},
		{
			name: "OnStopExecutedError",
			give: &OnStopExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			want: "[Fx] HOOK OnStop		hook.onStart1 called by bytes.NewBuffer failed in 0s: some error\n",
		},
		{
			name: "OnStopExecutedError/rich error",
			give: &OnStopExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          &richError{},
			},
			want: "[Fx] HOOK OnStop		hook.onStart1 called by bytes.NewBuffer failed in 0s: rich error\n",
		},
		{
			name: "OnStopExecuted",
			give: &OnStopExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			want: "[Fx] HOOK OnStop		hook.onStart1 called by bytes.NewBuffer ran successfully in 3ms\n",
		},
		{
			name: "OnStartExecutedError",
			give: &OnStartExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			want: "[Fx] HOOK OnStart		hook.onStart1 called by bytes.NewBuffer failed in 0s: some error\n",
		},
		{
			name: "OnStartExecutedError/rich error",
			give: &OnStartExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          &richError{},
			},
			want: "[Fx] HOOK OnStart		hook.onStart1 called by bytes.NewBuffer failed in 0s: rich error\n",
		},
		{
			name: "OnStartExecuted",
			give: &OnStartExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			want: "[Fx] HOOK OnStart		hook.onStart1 called by bytes.NewBuffer ran successfully in 3ms\n",
		},
		{
			name: "ProvideError",
			give: &Provided{Err: errors.New("some error")},
			want: "[Fx] Error after options were applied: some error\n",
		},
		{
			name: "ProvideError/rich error",
			give: &Provided{Err: &richError{}},
			want: "[Fx] Error after options were applied: rich error\n",
		},
		{
			name: "Supplied",
			give: &Supplied{TypeName: "*bytes.Buffer"},
			want: "[Fx] SUPPLY	*bytes.Buffer\n",
		},
		{
			name: "Supplied with module",
			give: &Supplied{TypeName: "*bytes.Buffer", ModuleName: "myModule"},
			want: "[Fx] SUPPLY	*bytes.Buffer from module \"myModule\"\n",
		},
		{
			name: "SuppliedError",
			give: &Supplied{TypeName: "*bytes.Buffer", Err: errors.New("great sadness")},
			want: "[Fx] ERROR	Failed to supply *bytes.Buffer: great sadness\n",
		},
		{
			name: "SuppliedError/rich error",
			give: &Supplied{TypeName: "*bytes.Buffer", Err: &richError{}},
			want: "[Fx] ERROR	Failed to supply *bytes.Buffer: rich error\n",
		},
		{
			name: "Provided",
			give: &Provided{
				ConstructorName: "bytes.NewBuffer()",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] PROVIDE	*bytes.Buffer <= bytes.NewBuffer()\n",
		},
		{
			name: "Provided with module",
			give: &Provided{
				ConstructorName: "bytes.NewBuffer()",
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] PROVIDE	*bytes.Buffer <= bytes.NewBuffer() from module \"myModule\"\n",
		},
		{
			name: "Replaced",
			give: &Replaced{
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] REPLACE	*bytes.Buffer\n",
		},
		{
			name: "Replaced with module",
			give: &Replaced{
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] REPLACE	*bytes.Buffer from module \"myModule\"\n",
		},
		{
			name: "ReplacedError",
			give: &Replaced{Err: errors.New("some error")},
			want: "[Fx] ERROR	Failed to replace: some error\n",
		},
		{
			name: "Decorated",
			give: &Decorated{
				DecoratorName:   "bytes.NewBuffer()",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] DECORATE	*bytes.Buffer <= bytes.NewBuffer()\n",
		},
		{
			name: "Decorated with module",
			give: &Decorated{
				DecoratorName:   "bytes.NewBuffer()",
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			want: "[Fx] DECORATE	*bytes.Buffer <= bytes.NewBuffer() from module \"myModule\"\n",
		},
		{
			name: "DecorateError",
			give: &Decorated{Err: errors.New("some error")},
			want: "[Fx] Error after options were applied: some error\n",
		},
		{
			name: "DecorateError/rich error",
			give: &Decorated{Err: &richError{}},
			want: "[Fx] Error after options were applied: rich error\n",
		},
		{
			name: "Invoking",
			give: &Invoking{FunctionName: "bytes.NewBuffer()"},
			want: "[Fx] INVOKE		bytes.NewBuffer()\n",
		},
		{
			name: "Invoking with module",
			give: &Invoking{
				FunctionName: "bytes.NewBuffer()",
				ModuleName:   "myModule",
			},
			want: "[Fx] INVOKE		bytes.NewBuffer() from module \"myModule\"\n",
		},
		{
			name: "Invoked/Error",
			give: &Invoked{
				FunctionName: "bytes.NewBuffer()",
				Err:          errors.New("some error"),
				Trace:        "foo()\n\tbar/baz.go:42\n",
			},
			want: joinLines(
				"[Fx] ERROR		fx.Invoke(bytes.NewBuffer()) called from:",
				"foo()",
				"	bar/baz.go:42",
				"Failed: some error",
			),
		},
		{
			name: "Invoked/Error/rich",
			give: &Invoked{
				FunctionName: "bytes.NewBuffer()",
				Err:          &richError{},
				Trace:        "foo()\n\tbar/baz.go:42\n",
			},
			want: joinLines(
				"[Fx] ERROR		fx.Invoke(bytes.NewBuffer()) called from:",
				"foo()",
				"	bar/baz.go:42",
				"Failed: rich error",
			),
		},
		{
			name: "StartError",
			give: &Started{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to start: some error\n",
		},
		{
			name: "StartError/rich error",
			give: &Started{Err: &richError{}},
			want: "[Fx] ERROR		Failed to start: rich error\n",
		},
		{
			name: "Stopping",
			give: &Stopping{Signal: os.Interrupt},
			want: "[Fx] INTERRUPT\n",
		},
		{
			name: "Stopped",
			give: &Stopped{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to stop cleanly: some error\n",
		},
		{
			name: "Stopped/rich error",
			give: &Stopped{Err: &richError{}},
			want: "[Fx] ERROR		Failed to stop cleanly: rich error\n",
		},
		{
			name: "RollingBack",
			give: &RollingBack{StartErr: errors.New("some error")},
			want: "[Fx] ERROR		Start failed, rolling back: some error\n",
		},
		{
			name: "RollingBack/rich error",
			give: &RollingBack{StartErr: &richError{}},
			want: "[Fx] ERROR		Start failed, rolling back: rich error\n",
		},
		{
			name: "RolledBack",
			give: &RolledBack{Err: errors.New("some error")},
			want: "[Fx] ERROR		Couldn't roll back cleanly: some error\n",
		},
		{
			name: "RolledBack/rich error",
			give: &RolledBack{Err: &richError{}},
			want: "[Fx] ERROR		Couldn't roll back cleanly: rich error\n",
		},
		{
			name: "Started",
			give: &Started{},
			want: "[Fx] RUNNING\n",
		},
		{
			name: "CustomLoggerError",
			give: &LoggerInitialized{Err: errors.New("great sadness")},
			want: "[Fx] ERROR		Failed to initialize custom logger: great sadness\n",
		},
		{
			name: "CustomLoggerError/rich error",
			give: &LoggerInitialized{Err: &richError{}},
			want: "[Fx] ERROR		Failed to initialize custom logger: rich error\n",
		},
		{
			name: "LoggerInitialized",
			give: &LoggerInitialized{ConstructorName: "go.uber.org/fx/fxevent.TestConsoleLogger.func1()"},
			want: "[Fx] LOGGER	Initialized custom logger from go.uber.org/fx/fxevent.TestConsoleLogger.func1()\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			(&ConsoleLogger{W: &buff}).LogEvent(tt.give)

			assert.Equal(t, tt.want, buff.String())
		})
	}
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
