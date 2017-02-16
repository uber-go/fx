// Copyright (c) 2017 Uber Technologies, Inc.
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

package ulog

import (
	"context"
	stderr "errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx/internal"
	"go.uber.org/fx/ulog/sentry"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber-go/zap"
	"github.com/uber-go/zap/spy"
	"github.com/uber/jaeger-client-go"
)

type baseLogger struct {
	sh  *sentry.Hook
	log zap.Logger
}

// Log is the UberFx wrapper for underlying logging service
type Log interface {
	// With creates a child logger with the provided parameters as key value pairs
	// ulog uses uber-go/zap library as its child logger which needs pairs of key value objects
	// in the form of zap.Fields(key, value). ulog performs field conversion from
	// supplied keyVals pair to zap.Fields format.
	//
	// **IMPORTANT**: With should never be used if the resulting logger
	// object is not being retained. If you need to add some context to
	// a logging message, use the Error, Info, etc. functions
	// and pass in additional interface{} pairs for logging.
	With(keyVals ...interface{}) Log

	// Check returns a zap.CheckedMessage if logging a message at the specified level is enabled.
	Check(level zap.Level, message string) *zap.CheckedMessage

	// Typed returns underlying logger implementation (zap.Logger) to get around the ulog.Log interface
	Typed() zap.Logger

	// Log at the provided zap.Level with message, and a sequence of parameters as key value pairs
	Log(level zap.Level, message string, keyVals ...interface{})

	// Debug logs at Debug level with message, and parameters as key value pairs
	Debug(message string, keyVals ...interface{})

	// Info logs at Info level with message, and parameters as key value pairs
	Info(message string, keyVals ...interface{})

	// Warn ogs at Warn level with message, and parameters as key value pairs
	Warn(message string, keyVals ...interface{})

	// Error logs at Error level with message, and parameters as key value pairs
	Error(message string, keyVals ...interface{})

	// Panic logs at Panic level with message, and parameters as key value pairs
	Panic(message string, keyVals ...interface{})

	// Fatal logs at Fatal level with message, and parameters as key value pairs
	Fatal(message string, keyVals ...interface{})

	// DPanic logs at Debug level (Fatal for development) with message, and parameters as key value pairs
	DPanic(message string, keyVals ...interface{})
}

var (
	_setupMu sync.RWMutex
	_std     = defaultLogger()
)

func defaultLogger() *baseLogger {
	return &baseLogger{
		log: zap.New(zap.NewJSONEncoder()),
	}
}

// TestingLogger returns basic logger and underlying sink for testing the messages
// WithInMemoryLogger testing helper can also be used to test actual outputted
// JSON bytes
func TestingLogger() (Log, *spy.Sink) {
	log, sink := spy.New()
	return &baseLogger{
		log: log,
	}, sink
}

func logger() Log {
	_setupMu.RLock()
	defer _setupMu.RUnlock()

	return &baseLogger{
		log: _std.log,
	}
}

// SetLogger sets configured logger at the start of the service
func SetLogger(log Log) {
	_setupMu.Lock()
	defer _setupMu.Unlock()

	// Log and _std log implements zap.Logger with set of predefined fields,
	// so we require users to use the configured logger. The Zap implementation however
	// can be overridden by log.SetLogger(zap.Logger)
	_std = log.(*baseLogger)
}

// Logger is the context based logger
func Logger(ctx context.Context) Log {
	if ctx == nil {
		panic("logger requires a context that cannot be nil")
	}
	log := ctx.Value(internal.ContextLogger)
	if log != nil {
		return log.(Log)
	}
	return logger()
}

// NewLogContext sets the context with the context aware logger
func NewLogContext(ctx context.Context, log Log) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if log != nil {
		return context.WithValue(ctx, internal.ContextLogger, log)
	}
	return context.WithValue(ctx, internal.ContextLogger, logger())
}

// WithTracingAware returns a new context with a context-aware logger
func WithTracingAware(ctx context.Context, span opentracing.Span) context.Context {
	// Note that opentracing.Tracer does not expose the tracer id
	// We only inject tracing information for jaeger.Tracer
	logger := Logger(ctx)
	if jSpanCtx, ok := span.Context().(jaeger.SpanContext); ok {
		logger = logger.With(
			"traceID", jSpanCtx.TraceID(), "spanID", jSpanCtx.SpanID(),
		)
	}
	return context.WithValue(ctx, internal.ContextLogger, logger)
}

// Typed returns underneath zap implementation for use
func (l *baseLogger) Typed() zap.Logger {
	return l.log
}

func (l *baseLogger) With(keyVals ...interface{}) Log {
	var sh *sentry.Hook
	if l.sh != nil {
		sh = l.sh.Copy()
		sh.AppendFields(keyVals...)
	}

	return &baseLogger{
		log: l.log.With(l.fieldsConversion(keyVals...)...),
		sh:  sh,
	}
}

func (l *baseLogger) Check(level zap.Level, expr string) *zap.CheckedMessage {
	return l.log.Check(level, expr)
}

func (l *baseLogger) Debug(message string, keyVals ...interface{}) {
	l.Log(zap.DebugLevel, message, keyVals...)
}

func (l *baseLogger) Info(message string, keyVals ...interface{}) {
	l.Log(zap.InfoLevel, message, keyVals...)
}

func (l *baseLogger) Warn(message string, keyVals ...interface{}) {
	l.Log(zap.WarnLevel, message, keyVals...)
}

func (l *baseLogger) Error(message string, keyVals ...interface{}) {
	l.Log(zap.ErrorLevel, message, keyVals...)
}

func (l *baseLogger) Panic(message string, keyVals ...interface{}) {
	l.Log(zap.PanicLevel, message, keyVals...)
}

func (l *baseLogger) Fatal(message string, keyVals ...interface{}) {
	l.Log(zap.FatalLevel, message, keyVals...)
}

func (l *baseLogger) DPanic(message string, keyVals ...interface{}) {
	l.log.DPanic(message, l.fieldsConversion(keyVals...)...)
}

func (l *baseLogger) Log(lvl zap.Level, message string, keyVals ...interface{}) {
	if cm := l.Check(lvl, message); cm.OK() {
		cm.Write(l.fieldsConversion(keyVals...)...)
	}
	if l.sh != nil {
		l.sh.CheckAndFire(lvl, message, keyVals...)
	}
}

type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

func (l *baseLogger) fieldsConversion(keyVals ...interface{}) []zap.Field {
	if len(keyVals)%2 != 0 {
		return []zap.Field{zap.Error(stderr.New("expected even number of arguments"))}
	}

	fields := make([]zap.Field, 0, len(keyVals)/2)
	for idx := 0; idx < len(keyVals); idx += 2 {
		if key, ok := keyVals[idx].(string); ok {
			switch value := keyVals[idx+1].(type) {
			case bool:
				fields = append(fields, zap.Bool(key, value))
			case float64:
				fields = append(fields, zap.Float64(key, value))
			case int:
				fields = append(fields, zap.Int(key, value))
			case int64:
				fields = append(fields, zap.Int64(key, value))
			case uint:
				fields = append(fields, zap.Uint(key, value))
			case uint64:
				fields = append(fields, zap.Uint64(key, value))
			case uintptr:
				fields = append(fields, zap.Uintptr(key, value))
			case string:
				fields = append(fields, zap.String(key, value))
			case time.Time:
				fields = append(fields, zap.Time(key, value))
			case time.Duration:
				fields = append(fields, zap.Duration(key, value))
			case zap.LogMarshaler:
				fields = append(fields, zap.Marshaler(key, value))
			case fmt.Stringer:
				fields = append(fields, zap.Stringer(key, value))
			case stackTracer:
				fields = append(fields,
					zap.String("stacktrace", fmt.Sprintf("%+v", value.StackTrace())),
					zap.Error(value))
			case error:
				fields = append(fields, zap.Error(value))
			default:
				fields = append(fields, zap.Object(key, value))
			}
		}
	}
	return fields
}
