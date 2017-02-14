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
	stderr "errors"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx/ulog/sentry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type baseLogger struct {
	sh  *sentry.Hook
	log *zap.Logger
}

// Log is the UberFx wrapper for underlying logging service
type Log interface {
	// With creates a child logger with the provided parameters as key value pairs
	// ulog uses uber-go/zap library as its child logger which needs pairs of key value objects
	// in the form of zapcore.Fields(key, value). ulog performs field conversion from
	// supplied keyVals pair to zapcore.Fields format.
	//
	// **IMPORTANT**: With should never be used if the resulting logger
	// object is not being retained. If you need to add some context to
	// a logging message, use the Error, Info, etc. functions
	// and pass in additional interface{} pairs for logging.
	With(keyVals ...interface{}) Log

	// Check returns a zap.CheckedEntry if logging a message at the specified level is enabled.
	Check(level zapcore.Level, message string) *zapcore.CheckedEntry

	// Typed returns underlying logger implementation (zap.Logger) to get around the ulog.Log interface
	Typed() *zap.Logger

	// Log at the provided zapcore.Level with message, and a sequence of parameters as key value pairs
	Log(level zapcore.Level, message string, keyVals ...interface{})

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

var _std Log

func init() {
	// TODO(pedge): this sucks, zap now has a no-op logger as the default as well
	// it would be better if we forced the logger to be setup, this panic is not good
	log, err := defaultLogger()
	if err != nil {
		panic(err.Error())
	}
	_std = log
}

func defaultLogger() (Log, error) {
	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &baseLogger{
		log: log,
	}, nil
}

// TestingLogger returns basic logger and underlying sink for testing the messages
// WithInMemoryLogger testing helper can also be used to test actual outputted
// JSON bytes
/*
func TestingLogger() (Log, *spy.Sink) {
	log, sink := spy.New()
	return &baseLogger{
		log: log,
	}, sink
}
*/

// Logger returns the package-level logger configured for the service
// TODO:(at) Remove Logger() call, _std and defaultLogger() access in ulog
func Logger() Log {
	return &baseLogger{
		log: _std.(*baseLogger).log,
	}
}

// Typed returns underneath zap implementation for use
func (l *baseLogger) Typed() *zap.Logger {
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

func (l *baseLogger) Check(level zapcore.Level, expr string) *zapcore.CheckedEntry {
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

func (l *baseLogger) Log(lvl zapcore.Level, message string, keyVals ...interface{}) {
	if cm := l.Check(lvl, message); cm != nil {
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

func (l *baseLogger) fieldsConversion(keyVals ...interface{}) []zapcore.Field {
	if len(keyVals)%2 != 0 {
		return []zapcore.Field{zap.Error(stderr.New("expected even number of arguments"))}
	}

	fields := make([]zapcore.Field, 0, len(keyVals)/2)
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
			case fmt.Stringer:
				fields = append(fields, zap.Stringer(key, value))
			case stackTracer:
				fields = append(fields,
					zap.String("stacktrace", fmt.Sprintf("%+v", value.StackTrace())),
					zap.Error(value))
			case error:
				fields = append(fields, zap.Error(value))
			default:
				fields = append(fields, zap.Any(key, value))
			}
		}
	}
	return fields
}
