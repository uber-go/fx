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

//go:build go1.21

package fxevent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type slogObservableEntry struct {
	record slog.Record
}

func (s slogObservableEntry) unwrap(attr slog.Attr, out map[string]interface{}) {
	anyAttr := attr.Value.Any()

	sliceAttr, ok := anyAttr.([]slog.Attr)
	if !ok {
		out[attr.Key] = anyAttr
		return
	}

	sliceAttrValues := make([]any, len(sliceAttr))
	for i, iter := range sliceAttr {
		sliceAttrValues[i] = iter.Value.Any()
	}

	out[attr.Key] = sliceAttrValues
}

func (s slogObservableEntry) ContextMap() map[string]interface{} {
	contextMap := map[string]interface{}{}

	s.record.Attrs(func(a slog.Attr) bool {
		s.unwrap(a, contextMap)
		return true
	})
	return contextMap
}

type slogObservableLogger struct {
	level   slog.Level
	entries []slogObservableEntry
	attrs   []slog.Attr
}

func (s *slogObservableLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return int(s.level) <= int(level)
}

func (s *slogObservableLogger) Handle(ctx context.Context, record slog.Record) error {
	s.entries = append(s.entries, slogObservableEntry{record})
	return nil
}

func (s *slogObservableLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogObservableLogger{
		level:   s.level,
		entries: s.entries,
		attrs:   append(s.attrs, attrs...),
	}
}

func (s *slogObservableLogger) WithGroup(name string) slog.Handler {
	return s
}

func (s *slogObservableLogger) TakeAll() []slogObservableEntry {
	return s.entries
}

func newSlogObservableLogger(level slog.Level) (*slog.Logger, *slogObservableLogger) {
	handler := &slogObservableLogger{level: level}
	return slog.New(handler), handler
}

func TestSlogLogger(t *testing.T) {
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
			name: "OnStopExecuted/Error",
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
			name: "OnStartExecuted/Error",
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
			name: "Supplied",
			give: &Supplied{
				TypeName:    "*bytes.Buffer",
				StackTrace:  []string{"main.main", "runtime.main"},
				ModuleTrace: []string{"main.main"},
			},
			wantMessage: "supplied",
			wantFields: map[string]interface{}{
				"type":        "*bytes.Buffer",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
			},
		},
		{
			name: "Supplied/Error",
			give: &Supplied{
				TypeName:    "*bytes.Buffer",
				StackTrace:  []string{"main.main", "runtime.main"},
				ModuleTrace: []string{"main.main"},
				Err:         someError,
			},
			wantMessage: "error encountered while applying options",
			wantFields: map[string]interface{}{
				"type":        "*bytes.Buffer",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"error":       "some error",
			},
		},
		{
			name: "Provide",
			give: &Provided{
				ConstructorName: "bytes.NewBuffer()",
				StackTrace:      []string{"main.main", "runtime.main"},
				ModuleTrace:     []string{"main.main"},
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
				Private:         false,
			},
			wantMessage: "provided",
			wantFields: map[string]interface{}{
				"constructor": "bytes.NewBuffer()",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"type":        "*bytes.Buffer",
				"module":      "myModule",
			},
		},
		{
			name: "PrivateProvide",
			give: &Provided{
				ConstructorName: "bytes.NewBuffer()",
				StackTrace:      []string{"main.main", "runtime.main"},
				ModuleTrace:     []string{"main.main"},
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
				Private:         true,
			},
			wantMessage: "provided",
			wantFields: map[string]interface{}{
				"constructor": "bytes.NewBuffer()",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"type":        "*bytes.Buffer",
				"module":      "myModule",
				"private":     true,
			},
		},
		{
			name: "Provide/Error",
			give: &Provided{
				StackTrace:  []string{"main.main", "runtime.main"},
				ModuleTrace: []string{"main.main"},
				Err:         someError,
			},
			wantMessage: "error encountered while applying options",
			wantFields: map[string]interface{}{
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"error":       "some error",
			},
		},
		{
			name: "Replace",
			give: &Replaced{
				ModuleName:      "myModule",
				StackTrace:      []string{"main.main", "runtime.main"},
				ModuleTrace:     []string{"main.main"},
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			wantMessage: "replaced",
			wantFields: map[string]interface{}{
				"type":        "*bytes.Buffer",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"module":      "myModule",
			},
		},
		{
			name: "Replace/Error",
			give: &Replaced{
				StackTrace:  []string{"main.main", "runtime.main"},
				ModuleTrace: []string{"main.main"},
				Err:         someError,
			},

			wantMessage: "error encountered while replacing",
			wantFields: map[string]interface{}{
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"error":       "some error",
			},
		},
		{
			name: "Decorate",
			give: &Decorated{
				DecoratorName:   "bytes.NewBuffer()",
				StackTrace:      []string{"main.main", "runtime.main"},
				ModuleTrace:     []string{"main.main"},
				ModuleName:      "myModule",
				OutputTypeNames: []string{"*bytes.Buffer"},
			},
			wantMessage: "decorated",
			wantFields: map[string]interface{}{
				"decorator":   "bytes.NewBuffer()",
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"type":        "*bytes.Buffer",
				"module":      "myModule",
			},
		},
		{
			name: "Decorate/Error",
			give: &Decorated{
				StackTrace:  []string{"main.main", "runtime.main"},
				ModuleTrace: []string{"main.main"},
				Err:         someError,
			},
			wantMessage: "error encountered while applying options",
			wantFields: map[string]interface{}{
				"stacktrace":  []interface{}{"main.main", "runtime.main"},
				"moduletrace": []interface{}{"main.main"},
				"error":       "some error",
			},
		},
		{
			name:        "BeforeRun",
			give:        &BeforeRun{Name: "bytes.NewBuffer()", Kind: "constructor"},
			wantMessage: "before run",
			wantFields: map[string]interface{}{
				"name":    "bytes.NewBuffer()",
				"kind":    "constructor",
			},
		},
		{
			name:        "Run",
			give:        &Run{Name: "bytes.NewBuffer()", Kind: "constructor", Runtime: 3 * time.Millisecond},
			wantMessage: "run",
			wantFields: map[string]interface{}{
				"name":    "bytes.NewBuffer()",
				"kind":    "constructor",
				"runtime": "3ms",
			},
		},
		{
			name: "Run with module",
			give: &Run{
				Name:       "bytes.NewBuffer()",
				Kind:       "constructor",
				ModuleName: "myModule",
				Runtime:    3 * time.Millisecond,
			},
			wantMessage: "run",
			wantFields: map[string]interface{}{
				"name":    "bytes.NewBuffer()",
				"kind":    "constructor",
				"module":  "myModule",
				"runtime": "3ms",
			},
		},
		{
			name: "Run/Error",
			give: &Run{
				Name: "bytes.NewBuffer()",
				Kind: "constructor",
				Err:  someError,
			},
			wantMessage: "error returned",
			wantFields: map[string]interface{}{
				"name":  "bytes.NewBuffer()",
				"kind":  "constructor",
				"error": "some error",
			},
		},
		{
			name:        "Invoking/Success",
			give:        &Invoking{ModuleName: "myModule", FunctionName: "bytes.NewBuffer()"},
			wantMessage: "invoking",
			wantFields: map[string]interface{}{
				"function": "bytes.NewBuffer()",
				"module":   "myModule",
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
			name:        "Start/Error",
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
			name:        "Stopped/Error",
			give:        &Stopped{Err: someError},
			wantMessage: "stop failed",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RollingBack/Error",
			give:        &RollingBack{StartErr: someError},
			wantMessage: "start failed, rolling back",
			wantFields: map[string]interface{}{
				"error": "some error",
			},
		},
		{
			name:        "RolledBack/Error",
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
			name:        "LoggerInitialized/Error",
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

	t.Run("debug observer, log at default (info)", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				core, observedLogs := newSlogObservableLogger(slog.LevelDebug)
				(&SlogLogger{Logger: core}).LogEvent(tt.give)

				logs := observedLogs.TakeAll()
				require.Len(t, logs, 1)
				got := logs[0]

				assert.Equal(t, tt.wantMessage, got.record.Message)
				assert.Equal(t, tt.wantFields, got.ContextMap())
			})
		}
	})

	t.Run("info observer, log at debug", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				core, observedLogs := newSlogObservableLogger(slog.LevelInfo)
				l := &SlogLogger{Logger: core}
				l.UseLogLevel(slog.LevelDebug)
				l.LogEvent(tt.give)

				logs := observedLogs.TakeAll()
				// logs are not visible unless they are errors
				if strings.HasSuffix(tt.name, "/Error") {
					require.Len(t, logs, 1)
					got := logs[0]
					assert.Equal(t, tt.wantMessage, got.record.Message)
					assert.Equal(t, tt.wantFields, got.ContextMap())
				} else {
					require.Len(t, logs, 0)
				}
			})
		}
	})

	t.Run("info observer, log/error at debug", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				core, observedLogs := newSlogObservableLogger(slog.LevelInfo)
				l := &SlogLogger{Logger: core}
				l.UseLogLevel(slog.LevelDebug)
				l.UseErrorLevel(slog.LevelDebug)
				l.LogEvent(tt.give)

				logs := observedLogs.TakeAll()
				require.Len(t, logs, 0, "no logs should be visible")
			})
		}
	})

	t.Run("test setting log levels", func(t *testing.T) {
		levels := []slog.Level{
			slog.LevelError,
			slog.LevelDebug,
			slog.LevelWarn,
			slog.LevelInfo,
		}

		for _, level := range levels {
			core, observedLogs := newSlogObservableLogger(level)
			logger := &SlogLogger{Logger: core}
			logger.UseLogLevel(level)
			func() {
				defer func() {
					recover()
				}()
				logger.LogEvent(&OnStartExecuting{
					FunctionName: "hook.onStart",
					CallerName:   "bytes.NewBuffer",
				})
			}()
			logs := observedLogs.TakeAll()
			require.Len(t, logs, 1)
		}
	})

	t.Run("test setting error log levels", func(t *testing.T) {
		levels := []slog.Level{
			slog.LevelError,
			slog.LevelDebug,
			slog.LevelWarn,
			slog.LevelInfo,
		}

		for _, level := range levels {
			core, observedLogs := newSlogObservableLogger(level)
			logger := &SlogLogger{Logger: core}
			logger.UseErrorLevel(level)
			func() {
				defer func() {
					recover()
				}()
				logger.LogEvent(&OnStopExecuted{
					FunctionName: "hook.onStart1",
					CallerName:   "bytes.NewBuffer",
					Err:          fmt.Errorf("some error"),
				})
			}()
			logs := observedLogs.TakeAll()
			require.Len(t, logs, 1)
		}
	})
}
