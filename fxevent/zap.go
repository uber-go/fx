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

	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/zap"
)

// ZapLogger is an Fx event logger that logs events to Zap.
type ZapLogger struct {
	Logger *zap.Logger
}

var _ Logger = (*ZapLogger)(nil)

// LogEvent logs the given event to the provided Zap logger.
func (l *ZapLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case *LifecycleHookExecuting:
		l.Logger.Info("hook executing",
			zap.String("method", e.Method),
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *LifecycleHookExecuted:
		if e.Err != nil {
			l.Logger.Info("hook execute failed",
				zap.String("method", e.Method),
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.Logger.Info("hook executed",
				zap.String("method", e.Method),
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *ProvideError:
		l.Logger.Error("error encountered while applying options",
			zap.Error(e.Err))
	case *Supply:
		l.Logger.Info("supplying", zap.String("type", e.TypeName))
	case *Provide:
		for _, rtype := range e.OutputTypeNames {
			l.Logger.Info("providing",
				zap.String("constructor", fxreflect.FuncName(e.Constructor)),
				zap.String("type", rtype),
			)
		}
	case *Invoke:
		l.Logger.Info("invoke",
			zap.String("function", fxreflect.FuncName(e.Function)))
	case *InvokeError:
		l.Logger.Error("fx.Invoke failed",
			zap.Error(e.Err),
			zap.String("stack", e.Stacktrace),
			zap.String("function", fxreflect.FuncName(e.Function)))
	case *StartError:
		l.Logger.Error("failed to start", zap.Error(e.Err))
	case *StopSignal:
		l.Logger.Info("received signal",
			zap.String("signal", strings.ToUpper(e.Signal.String())))
	case *StopError:
		l.Logger.Error("failed to stop cleanly", zap.Error(e.Err))
	case *RollbackError:
		l.Logger.Error("could not rollback cleanly", zap.Error(e.Err))
	case *Rollback:
		l.Logger.Error("startup failed, rolling back", zap.Error(e.StartErr))
	case *Running:
		l.Logger.Info("running")
	case *CustomLoggerError:
		l.Logger.Error("error constructing logger", zap.Error(e.Err))
	case *CustomLogger:
		l.Logger.Info("installing custom fxevent.Logger",
			zap.String("function", fxreflect.FuncName(e.Function)))
	}
}
