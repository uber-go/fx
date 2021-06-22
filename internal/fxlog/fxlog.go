// Copyright (c) 2020 Uber Technologies, Inc.
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

package fxlog

import (
	"strings"

	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines interface used for logging.
type Logger interface {
	// LogEvent is called when a logging event is emitted.
	LogEvent(Event)
}

type EventLogger interface {
	LogEvent(Event)
}

type Event interface {
	// event()
}

// LifecycleOnStartEvent is emitted for whenever an OnStart hook is executed
type LifecycleOnStartEvent struct {
	Caller string
}

// LifecycleOnStopEvent is emitted for whenever an OnStart hook is executed
type LifecycleOnStopEvent struct {
	Caller string
}
// ApplyOptionsError is emitted whenever there is an error applying options.
type ApplyOptionsError struct {
	Err error
}

// SupplyEvent is emitted whenever a Provide was called with a constructor provided
// by fx.Supply.
type SupplyEvent struct{
	Constructor interface{}
}

// ProvideEvent is emitted whenever Provide was called and is not provided by fx.Supply.
type ProvideEvent struct {
	Constructor interface{}
}

// InvokeEvent is emitted whenever a function is invoked.
type InvokeEvent struct {
	Function interface{}
}

// InvokeFailedEvent is emitted when fx.Invoke has failed.
type InvokeFailedEvent struct {
	Function interface{}
	Err error
	Stack fxreflect.Stack
}

// StartFailureError is emitted right before exiting after failing to start.
type StartFailureError struct { Err error}

// StopSignalEvent is emitted whenever application receives a signal after
// starting the application.
type StopSignalEvent struct{ Signal string }

// StopErrorEvent is emitted whenever we fail to stop cleanly.
type StopErrorEvent struct{ Err error }

// StartErrorEvent is emitted whenever a service fails to start.
type StartErrorEvent struct{ Err error }

// StartRollbackError is emitted whenever we fail to rollback cleanly after
// a start error.
type StartRollbackError struct {Err error}

// RunningEvent is emitted whenever an application is started successfully.
type RunningEvent struct {}

var _ Logger = (*zapLogger)(nil)

type zapLogger struct {
	logger *zap.Logger
}

func (l *zapLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case LifecycleOnStartEvent:
		l.logger.Info("starting", zap.String("caller", e.Caller))
	case ApplyOptionsError:
		l.logger.Error("error encountered while applying options", zap.Error(e.Err))
	case SupplyEvent:
		for _, rtype := range fxreflect.ReturnTypes(e.Constructor) {
			l.logger.Info("supplying",
				zap.String("constructor", fxreflect.FuncName(e.Constructor)),
				zap.String("type", rtype),
			)
		}
	case ProvideEvent:
		for _, rtype := range fxreflect.ReturnTypes(e.Constructor) {
			l.logger.Info("providing",
				zap.String("constructor", fxreflect.FuncName(e.Constructor)),
				zap.String("type", rtype),
			)
		}
	case InvokeEvent:
		l.logger.Info("invoke", zap.String("function", fxreflect.FuncName(e.Function)))
	case InvokeFailedEvent:
		l.logger.Error("fx.Invoke failed",
			zap.Error(e.Err),
			zap.String("stack", e.Stack.String()),
			zap.String("function", fxreflect.FuncName(e.Function)))
	case StartFailureError:
		l.logger.Info("failed to start", zap.Error(event.(StartFailureError).Err))
	case StopSignalEvent:
		l.logger.Info("received signal", zap.String("signal", strings.ToUpper(e.Signal)))
	case StopErrorEvent:
		l.logger.Error("failed to stop cleanly", zap.Error(event.(StopErrorEvent).Err))
	case StartRollbackError:
		l.logger.Error("could not rollback cleanly", zap.Error(event.(StartRollbackError).Err))
	case StartErrorEvent:
		l.logger.Error("startup failed, rolling back", zap.Error(event.(StartErrorEvent).Err))
	case RunningEvent:
		l.logger.Info("running")
	}
}

// DefaultLogger constructs a Logger out of io.Writer.
func DefaultLogger(ws zapcore.WriteSyncer) Logger {
	zcore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		ws,
		zap.NewAtomicLevel(),
	)
	log := zap.New(zcore)

	return &zapLogger{
		logger: log,
	}
}
