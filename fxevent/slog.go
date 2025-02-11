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
	"log/slog"
	"strconv"
	"strings"
)

var _ Logger = (*SlogLogger)(nil)

// SlogLogger an Fx event logger that logs events using a slog logger.
type SlogLogger struct {
	Logger *slog.Logger

	ctx        context.Context
	logLevel   slog.Level
	errorLevel *slog.Level
}

// UseContext sets the context that will be used when logging to slog.
func (l *SlogLogger) UseContext(ctx context.Context) {
	l.ctx = ctx
}

// UseLogLevel sets the level of non-error logs emitted by Fx to level.
func (l *SlogLogger) UseLogLevel(level slog.Level) {
	l.logLevel = level
}

// UseErrorLevel sets the level of error logs emitted by Fx to level.
func (l *SlogLogger) UseErrorLevel(level slog.Level) {
	l.errorLevel = &level
}

func (l *SlogLogger) filter(fields []any) []any {
	filtered := []any{}

	for _, field := range fields {
		if field, ok := field.(slog.Attr); ok {
			if _, ok := field.Value.Any().(slogFieldSkip); ok {
				continue
			}
		}

		filtered = append(filtered, field)
	}

	return filtered
}

func (l *SlogLogger) logEvent(msg string, fields ...any) {
	l.Logger.Log(l.ctx, l.logLevel, msg, l.filter(fields)...)
}

func (l *SlogLogger) logError(msg string, fields ...any) {
	lvl := slog.LevelError
	if l.errorLevel != nil {
		lvl = *l.errorLevel
	}

	l.Logger.Log(l.ctx, lvl, msg, l.filter(fields)...)
}

// LogEvent logs the given event to the provided Zap logger.
func (l *SlogLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case *OnStartExecuting:
		l.logEvent("OnStart hook executing",
			slog.String("callee", e.FunctionName),
			slog.String("caller", e.CallerName),
		)
	case *OnStartExecuted:
		if e.Err != nil {
			l.logError("OnStart hook failed",
				slog.String("callee", e.FunctionName),
				slog.String("caller", e.CallerName),
				slogErr(e.Err),
			)
		} else {
			l.logEvent("OnStart hook executed",
				slog.String("callee", e.FunctionName),
				slog.String("caller", e.CallerName),
				slog.String("runtime", e.Runtime.String()),
			)
		}
	case *OnStopExecuting:
		l.logEvent("OnStop hook executing",
			slog.String("callee", e.FunctionName),
			slog.String("caller", e.CallerName),
		)
	case *OnStopExecuted:
		if e.Err != nil {
			l.logError("OnStop hook failed",
				slog.String("callee", e.FunctionName),
				slog.String("caller", e.CallerName),
				slogErr(e.Err),
			)
		} else {
			l.logEvent("OnStop hook executed",
				slog.String("callee", e.FunctionName),
				slog.String("caller", e.CallerName),
				slog.String("runtime", e.Runtime.String()),
			)
		}
	case *Supplied:
		if e.Err != nil {
			l.logError("error encountered while applying options",
				slog.String("type", e.TypeName),
				slogStrings("moduletrace", e.ModuleTrace),
				slogStrings("stacktrace", e.StackTrace),
				slogMaybeModuleField(e.ModuleName),
				slogErr(e.Err))
		} else {
			l.logEvent("supplied",
				slog.String("type", e.TypeName),
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
			)
		}
	case *Provided:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("provided",
				slog.String("constructor", e.ConstructorName),
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
				slog.String("type", rtype),
				slogMaybeBool("private", e.Private),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				slogMaybeModuleField(e.ModuleName),
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogErr(e.Err))
		}
	case *Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("replaced",
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
				slog.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while replacing",
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
				slogErr(e.Err))
		}
	case *Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("decorated",
				slog.String("decorator", e.DecoratorName),
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
				slog.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				slogStrings("stacktrace", e.StackTrace),
				slogStrings("moduletrace", e.ModuleTrace),
				slogMaybeModuleField(e.ModuleName),
				slogErr(e.Err))
		}
	case *Run:
		if e.Err != nil {
			l.logError("error returned",
				slog.String("name", e.Name),
				slog.String("kind", e.Kind),
				slogMaybeModuleField(e.ModuleName),
				slogErr(e.Err),
			)
		} else {
			l.logEvent("run",
				slog.String("name", e.Name),
				slog.String("kind", e.Kind),
				slog.String("runtime", e.Runtime.String()),
				slogMaybeModuleField(e.ModuleName),
			)
		}
	case *Invoking:
		// Do not log stack as it will make logs hard to read.
		l.logEvent("invoking",
			slog.String("function", e.FunctionName),
			slogMaybeModuleField(e.ModuleName),
		)
	case *Invoked:
		if e.Err != nil {
			l.logError("invoke failed",
				slogErr(e.Err),
				slog.String("stack", e.Trace),
				slog.String("function", e.FunctionName),
				slogMaybeModuleField(e.ModuleName),
			)
		}
	case *Stopping:
		l.logEvent("received signal",
			slog.String("signal", strings.ToUpper(e.Signal.String())))
	case *Stopped:
		if e.Err != nil {
			l.logError("stop failed", slogErr(e.Err))
		}
	case *RollingBack:
		l.logError("start failed, rolling back", slogErr(e.StartErr))
	case *RolledBack:
		if e.Err != nil {
			l.logError("rollback failed", slogErr(e.Err))
		}
	case *Started:
		if e.Err != nil {
			l.logError("start failed", slogErr(e.Err))
		} else {
			l.logEvent("started")
		}
	case *LoggerInitialized:
		if e.Err != nil {
			l.logError("custom logger initialization failed", slogErr(e.Err))
		} else {
			l.logEvent("initialized custom fxevent.Logger", slog.String("function", e.ConstructorName))
		}
	}
}

type slogFieldSkip struct{}

func slogMaybeModuleField(name string) slog.Attr {
	if len(name) == 0 {
		return slog.Any("module", slogFieldSkip{})
	}
	return slog.String("module", name)
}

func slogMaybeBool(name string, b bool) slog.Attr {
	if !b {
		return slog.Any(name, slogFieldSkip{})
	}
	return slog.Bool(name, true)
}

func slogErr(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func slogStrings(key string, str []string) slog.Attr {
	attrs := make([]any, len(str))
	for i, val := range str {
		attrs[i] = slog.String(strconv.Itoa(i), val)
	}
	return slog.Group(key, attrs...)
}
