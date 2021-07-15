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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestZapLogger(t *testing.T) {
	t.Parallel()

	someError := errors.New("some error")

	tests := []struct {
		name        string
		give        Event
		wantMessage string
		wantFields  map[string]interface{}
	}{
		{
			name: "LifecycleHookExecuting",
			give: &LifecycleHookExecuting{
				Method:       "OnStop",
				FunctionName: "hook.onStop1",
				CallerName:   "bytes.NewBuffer",
			},
			wantMessage: "hook executing",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStop1",
				"method": "OnStop",
			},
		},
		{
			name: "LifecycleHookExecutedError",
			give: &LifecycleHookExecuted{
				Method:       "OnStart",
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			wantMessage: "hook execute failed",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStart1",
				"method": "OnStart",
				"error":  "some error",
			},
		},
		{
			name: "LifecycleHookExecuted",
			give: &LifecycleHookExecuted{
				Method:       "OnStart",
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			wantMessage: "hook executed",
			wantFields: map[string]interface{}{
				"caller":  "bytes.NewBuffer",
				"callee":  "hook.onStart1",
				"method":  "OnStart",
				"runtime": "3ms",
			},
		},
		{
			name:        "ProvideError",
			give:        &ProvideError{Err: someError},
			wantMessage: "error encountered while applying options",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Supply",
			give:        &Supply{TypeName: "*bytes.Buffer"},
			wantMessage: "supplying",
			wantFields: map[string]interface{}{
				"type": "*bytes.Buffer",
			},
		},
		{
			name:        "Provide",
			give:        &Provide{bytes.NewBuffer, []string{"*bytes.Buffer"}},
			wantMessage: "providing",
			wantFields: map[string]interface{}{
				"constructor": "bytes.NewBuffer()",
				"type":        "*bytes.Buffer",
			},
		},
		{
			name:        "Invoke",
			give:        &Invoke{bytes.NewBuffer},
			wantMessage: "invoke",
			wantFields: map[string]interface{}{
				"function": "bytes.NewBuffer()",
			},
		},
		{
			name:        "InvokeError",
			give:        &InvokeError{Function: bytes.NewBuffer, Err: someError},
			wantMessage: "fx.Invoke failed",
			wantFields: map[string]interface{}{
				"error":    "some error",
				"stack":    "",
				"function": "bytes.NewBuffer()",
			},
		},
		{
			name:        "StartError",
			give:        &StartError{Err: someError},
			wantMessage: "failed to start",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "StopSignal",
			give:        &StopSignal{Signal: os.Interrupt},
			wantMessage: "received signal",
			wantFields: map[string]interface{}{
				"signal": "INTERRUPT",
			},
		},
		{
			name:        "StopError",
			give:        &StopError{Err: someError},
			wantMessage: "failed to stop cleanly",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RollbackError",
			give:        &RollbackError{Err: someError},
			wantMessage: "could not rollback cleanly",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Rollback",
			give:        &Rollback{StartErr: someError},
			wantMessage: "startup failed, rolling back",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Running",
			give:        &Running{},
			wantMessage: "running",
			wantFields:  map[string]interface{}{},
		},
		{
			name:        "CustomLoggerError",
			give:        &CustomLoggerError{Err: someError},
			wantMessage: "error constructing logger",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "CustomLogger",
			give:        &CustomLogger{bytes.NewBuffer},
			wantMessage: "installing custom fxevent.Logger",
			wantFields: map[string]interface{}{
				"function": "bytes.NewBuffer()",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, observedLogs := observer.New(zap.DebugLevel)
			(&ZapLogger{Logger: zap.New(core)}).LogEvent(tt.give)

			logs := observedLogs.TakeAll()
			require.Len(t, logs, 1)
			got := logs[0]

			assert.Equal(t, tt.wantMessage, got.Message)
			assert.Equal(t, tt.wantFields, got.ContextMap())
		})
	}
}
