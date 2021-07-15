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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConsoleLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give Event
		want string
	}{
		{
			name: "LifecycleHookExecuting",
			give: &LifecycleHookExecuting{
				Method:       "OnStop",
				FunctionName: "hook.onStop1",
				CallerName:   "bytes.NewBuffer",
			},
			want: "[Fx] HOOK OnStop		hook.onStop1 executing (caller: bytes.NewBuffer)\n",
		},
		{
			name: "LifecycleHookExecutedError",
			give: &LifecycleHookExecuted{
				Method:       "OnStart",
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			want: "[Fx] HOOK OnStart		hook.onStart1 called by bytes.NewBuffer failed in 0s: some error\n",
		},
		{
			name: "LifecycleHookExecuted",
			give: &LifecycleHookExecuted{
				Method:       "OnStart",
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			want: "[Fx] HOOK OnStart		hook.onStart1 called by bytes.NewBuffer ran successfully in 3ms\n",
		},
		{
			name: "ProvideError",
			give: &ProvideError{Err: errors.New("some error")},
			want: "[Fx] Error after options were applied: some error\n",
		},
		{
			name: "Supply",
			give: &Supply{TypeName: "*bytes.Buffer"},
			want: "[Fx] SUPPLY	*bytes.Buffer\n",
		},
		{
			name: "Provide",
			give: &Provide{bytes.NewBuffer, []string{"*bytes.Buffer"}},
			want: "[Fx] PROVIDE	*bytes.Buffer <= bytes.NewBuffer()\n",
		},
		{
			name: "Invoke",
			give: &Invoke{bytes.NewBuffer},
			want: "[Fx] INVOKE		bytes.NewBuffer()\n",
		},
		{
			name: "InvokeError",
			give: &InvokeError{
				Function:   bytes.NewBuffer,
				Err:        errors.New("some error"),
				Stacktrace: "foo()\n\tbar/baz.go:42\n",
			},
			want: joinLines(
				"[Fx] fx.Invoke(bytes.NewBuffer()) called from:",
				"foo()",
				"	bar/baz.go:42",
				"Failed: some error",
			),
		},
		{
			name: "StartError",
			give: &StartError{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to start: some error\n",
		},
		{
			name: "StopSignal",
			give: &StopSignal{Signal: os.Interrupt},
			want: "[Fx] INTERRUPT\n",
		},
		{
			name: "StopError",
			give: &StopError{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to stop cleanly: some error\n",
		},
		{
			name: "RollbackError",
			give: &RollbackError{Err: errors.New("some error")},
			want: "[Fx] ERROR		Couldn't roll back cleanly: some error\n",
		},
		{
			name: "Rollback",
			give: &Rollback{StartErr: errors.New("some error")},
			want: "[Fx] ERROR		Start failed, rolling back: some error\n",
		},
		{
			name: "Running",
			give: &Running{},
			want: "[Fx] RUNNING\n",
		},
		{
			name: "CustomLoggerError",
			give: &CustomLoggerError{Err: errors.New("great sadness")},
			want: "[Fx] ERROR		Failed to construct custom logger: great sadness\n",
		},
		{
			name: "CustomLogger",
			give: &CustomLogger{
				Function: func() Logger { panic("should not run") },
			},
			want: "[Fx] LOGGER	Setting up custom logger from go.uber.org/fx/fxevent.TestConsoleLogger.func1()\n",
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
