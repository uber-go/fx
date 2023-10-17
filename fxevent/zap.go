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
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is an Fx event logger that logs events to Zap.
type ZapLogger struct {
	Logger *zap.Logger

	logLevel   zapcore.Level // default: zapcore.InfoLevel
	errorLevel *zapcore.Level
}

var _ Logger = (*ZapLogger)(nil)

// UseErrorLevel sets the level of error logs emitted by Fx to level.
func (l *ZapLogger) UseErrorLevel(level zapcore.Level) {
	l.errorLevel = &level
}

// UseLogLevel sets the level of non-error logs emitted by Fx to level.
func (l *ZapLogger) UseLogLevel(level zapcore.Level) {
	l.logLevel = level
}

func (l *ZapLogger) logEvent(msg string, fields ...zap.Field) {
	l.Logger.Log(l.logLevel, msg, fields...)
}

func (l *ZapLogger) logError(msg string, fields ...zap.Field) {
	lvl := zapcore.ErrorLevel
	if l.errorLevel != nil {
		lvl = *l.errorLevel
	}
	l.Logger.Log(lvl, msg, fields...)
}

// LogEvent logs the given event to the provided Zap logger.
func (l *ZapLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case *OnStartExecuting:
		l.logEvent("OnStart hook executing",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *OnStartExecuted:
		if e.Err != nil {
			l.logError("OnStart hook failed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("OnStart hook executed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *OnStopExecuting:
		l.logEvent("OnStop hook executing",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *OnStopExecuted:
		if e.Err != nil {
			l.logError("OnStop hook failed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("OnStop hook executed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *Supplied:
		if e.Err != nil {
			l.logError("error encountered while applying options",
				zap.String("type", e.TypeName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		} else {
			l.logEvent("supplied",
				zap.String("type", e.TypeName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
			)
		}
	case *Provided:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("provided",
				zap.String("constructor", e.ConstructorName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
				maybeBool("private", e.Private),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				moduleField(e.ModuleName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Error(e.Err))
		}
	case *Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("replaced",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while replacing",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		}
	case *Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("decorated",
				zap.String("decorator", e.DecoratorName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		}
	case *Run:
		if e.Err != nil {
			l.logError("error returned",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				moduleField(e.ModuleName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("run",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				moduleField(e.ModuleName),
			)
		}
	case *Invoking:
		// Do not log stack as it will make logs hard to read.
		l.logEvent("invoking",
			zap.String("function", e.FunctionName),
			moduleField(e.ModuleName),
		)
	case *Invoked:
		if e.Err != nil {
			l.logError("invoke failed",
				zap.Error(e.Err),
				zap.String("stack", e.Trace),
				zap.String("function", e.FunctionName),
				moduleField(e.ModuleName),
			)
		}
	case *Stopping:
		l.logEvent("received signal",
			zap.String("signal", strings.ToUpper(e.Signal.String())))
	case *Stopped:
		if e.Err != nil {
			l.logError("stop failed", zap.Error(e.Err))
		}
	case *RollingBack:
		l.logError("start failed, rolling back", zap.Error(e.StartErr))
	case *RolledBack:
		if e.Err != nil {
			l.logError("rollback failed", zap.Error(e.Err))
		}
	case *Started:
		if e.Err != nil {
			l.logError("start failed", zap.Error(e.Err))
		} else {
			l.logEvent("started")
		}
	case *LoggerInitialized:
		if e.Err != nil {
			l.logError("custom logger initialization failed", zap.Error(e.Err))
		} else {
			l.logEvent("initialized custom fxevent.Logger", zap.String("function", e.ConstructorName))
		}
	}
}

func moduleField(name string) zap.Field {
	if len(name) == 0 {
		return zap.Skip()
	}
	return zap.String("module", name)
}

func maybeBool(name string, b bool) zap.Field {
	if b {
		return zap.Bool(name, true)
	}
	return zap.Skip()
}
