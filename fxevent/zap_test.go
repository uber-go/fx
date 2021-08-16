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
			name: "OnStartExecuting",
			give: &OnStartExecuting{
				FunctionName: "hook.onStart",
				CallerName:   "bytes.NewBuffer",
			},
			wantMessage: "OnStart hook executing",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStart",
			},
		},
		{
			name: "OnStopExecuting",
			give: &OnStopExecuting{
				FunctionName: "hook.onStop1",
				CallerName:   "bytes.NewBuffer",
			},
			wantMessage: "OnStop hook executing",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStop1",
			},
		},
		{

			name: "OnStopExecutedError",
			give: &OnStopExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			wantMessage: "OnStop hook failed",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStart1",
				"error":  "some error",
			},
		},
		{
			name: "OnStopExecuted",
			give: &OnStopExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			wantMessage: "OnStop hook executed",
			wantFields: map[string]interface{}{
				"caller":  "bytes.NewBuffer",
				"callee":  "hook.onStart1",
				"runtime": "3ms",
			},
		},
		{

			name: "OnStartExecutedError",
			give: &OnStartExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Err:          fmt.Errorf("some error"),
			},
			wantMessage: "OnStart hook failed",
			wantFields: map[string]interface{}{
				"caller": "bytes.NewBuffer",
				"callee": "hook.onStart1",
				"error":  "some error",
			},
		},
		{
			name: "OnStartExecuted",
			give: &OnStartExecuted{
				FunctionName: "hook.onStart1",
				CallerName:   "bytes.NewBuffer",
				Runtime:      time.Millisecond * 3,
			},
			wantMessage: "OnStart hook executed",
			wantFields: map[string]interface{}{
				"caller":  "bytes.NewBuffer",
				"callee":  "hook.onStart1",
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
			name:        "Invoking/Success",
			give:        &Invoking{FunctionName: "bytes.NewBuffer()"},
			wantMessage: "invoking",
			wantFields: map[string]interface{}{
				"function": "bytes.NewBuffer()",
			},
		},
		{
			name:        "Invoked/Error",
			give:        &Invoked{FunctionName: "bytes.NewBuffer()", Err: someError},
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
			name:        "Stopping",
			give:        &Stopping{Signal: os.Interrupt},
			wantMessage: "received signal",
			wantFields: map[string]interface{}{
				"signal": "INTERRUPT",
			},
		},
		{
			name:        "Stopped",
			give:        &Stopped{Err: someError},
			wantMessage: "stop failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RollingBack",
			give:        &RollingBack{StartErr: someError},
			wantMessage: "start failed, rolling back",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RolledBackError",
			give:        &RolledBack{Err: someError},
			wantMessage: "rollback failed",
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
			give:        &LoggerInitialized{ConstructorName: "bytes.NewBuffer()"},
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
