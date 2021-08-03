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
			name:        "Supplied",
			give:        &Supplied{TypeName: "*bytes.Buffer"},
			wantMessage: "supplied",
			wantFields: map[string]interface{}{
				"type": "*bytes.Buffer",
			},
		},
		{
			name:        "SuppliedError",
			give:        &Supplied{TypeName: "*bytes.Buffer", Err: someError},
			wantMessage: "supplied",
			wantFields: map[string]interface{}{
				"type":  "*bytes.Buffer",
				"error": "some error",
			},
		},
		{
			name: "Provide",
			give: &Provided{
				ConstructorName: "bytes.NewBuffer()",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			wantMessage: "provided",
			wantFields: map[string]interface{}{
				"constructor": "bytes.NewBuffer()",
				"type":        "*bytes.Buffer",
			},
		},
		{
			name:        "Provide with Error",
			give:        &Provided{Err: someError},
			wantMessage: "error encountered while applying options",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Invoking",
			give:        &Invoking{FunctionName: "bytes.NewBuffer()"},
			wantMessage: "invoked",
			wantFields: map[string]interface{}{
				"function": "bytes.NewBuffer()",
			},
		},
		{
			name:        "InvokeError",
			give:        &Invoked{Function: bytes.NewBuffer, Err: someError},
			wantMessage: "invoke failed",
			wantFields: map[string]interface{}{
				"error":    "some error",
				"stack":    "",
				"function": "bytes.NewBuffer()",
			},
		},
		{
			name:        "StartError",
			give:        &Started{Err: someError},
			wantMessage: "start failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Stop",
			give:        &Stop{Signal: os.Interrupt},
			wantMessage: "received signal",
			wantFields: map[string]interface{}{
				"signal": "INTERRUPT",
			},
		},
		{
			name:        "StopError",
			give:        &Stop{Err: someError},
			wantMessage: "stop failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RollbackError",
			give:        &Rollback{Err: someError},
			wantMessage: "rollback failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Rollback",
			give:        &Rollback{StartErr: someError},
			wantMessage: "start failed, rolling back",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "Started",
			give:        &Started{},
			wantMessage: "started",
			wantFields:  map[string]interface{}{},
		},
		{
			name:        "LoggerInitialized Error",
			give:        &LoggerInitialized{Err: someError},
			wantMessage: "custom logger initialization failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "LoggerInitialized",
			give:        &LoggerInitialized{Constructor: bytes.NewBuffer, Err: nil},
			wantMessage: "initialized custom fxevent.Logger",
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
