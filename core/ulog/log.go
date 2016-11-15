// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"
	"time"

	"github.com/uber-go/zap"
)

type baselogger struct {
	log zap.Logger
}

// Log is the Uberfx wrapper for underlying logging service
type Log interface {

	// With creates a child logger with the provided parameters as key value pairs
	// ulog uses uber-go/zap library as its child logger which needs pairs of key value objects
	// in the form of zap.Fields(key, value). ulog performs field conversion from
	// supplied keyvals pair to zap.Fields format.
	With(keyvals ...interface{}) Log

	// Check returns a zap.CheckedMessage if logging a message at the specified level is enabled.
	Check(level zap.Level, message string) *zap.CheckedMessage

	// RawLogger returns underlying logger implementation (zap.Logger) to get around the ulog.Log interface
	RawLogger() zap.Logger

	// Log at the provided zap.Level with message, and a sequence of parameters as key value pairs
	Log(level zap.Level, message string, keyvals ...interface{})

	// Debug logs at Debug level with message, and parameters as key value pairs
	Debug(message string, keyvals ...interface{})

	// Info logs at Info level with message, and parameters as key value pairs
	Info(message string, keyvals ...interface{})

	// Warn ogs at Warn level with message, and parameters as key value pairs
	Warn(message string, keyvals ...interface{})

	// Error logs at Error level with message, and parameters as key value pairs
	Error(message string, keyvals ...interface{})

	// Panic logs at Panic level with message, and parameters as key value pairs
	Panic(message string, keyvals ...interface{})

	// Fatal logs at Fatal level with message, and parameters as key value pairs
	Fatal(message string, keyvals ...interface{})

	// DFatal logs at Debug level (Fatal for development) with message, and parameters as key value pairs
	DFatal(message string, keyvals ...interface{})
}

var _std = defaultLogger()

func defaultLogger() Log {
	return &baselogger{
		log: zap.New(zap.NewJSONEncoder()),
	}
}

// TODO: (at) Remove Logger() call, _std and defaultLogger() access in ulog
// Logger returns the package-level logger configured for the service
func Logger() Log {
	return &baselogger{
		log: _std.(*baselogger).log,
	}
}

// RawLogger returns underneath zap implementation for use
func (l *baselogger) RawLogger() zap.Logger {
	return l.log
}

func (l *baselogger) With(keyvals ...interface{}) Log {
	return &baselogger{
		log: l.log.With(l.fieldsConversion(keyvals...)...),
	}
}

func (l *baselogger) Check(level zap.Level, expr string) *zap.CheckedMessage {
	return l.log.Check(level, expr)
}

func (l *baselogger) Debug(message string, keyvals ...interface{}) {
	l.Log(zap.DebugLevel, message, keyvals...)
}

func (l *baselogger) Info(message string, keyvals ...interface{}) {
	l.Log(zap.InfoLevel, message, keyvals...)
}

func (l *baselogger) Warn(message string, keyvals ...interface{}) {
	l.Log(zap.WarnLevel, message, keyvals...)
}

func (l *baselogger) Error(message string, keyvals ...interface{}) {
	l.Log(zap.ErrorLevel, message, keyvals...)
}

func (l *baselogger) Panic(message string, keyvals ...interface{}) {
	l.Log(zap.PanicLevel, message, keyvals...)
}

func (l *baselogger) Fatal(message string, keyvals ...interface{}) {
	l.Log(zap.FatalLevel, message, keyvals...)
}

func (l *baselogger) DFatal(message string, keyvals ...interface{}) {
	l.log.DFatal(message, l.fieldsConversion(keyvals...)...)
}

func (l *baselogger) Log(lvl zap.Level, message string, keyvals ...interface{}) {
	if cm := l.Check(lvl, message); cm.OK() {
		cm.Write(l.fieldsConversion(keyvals...)...)
	}
}

func (l *baselogger) fieldsConversion(keyvals ...interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keyvals)/2)
	if len(keyvals)%2 != 0 {
		fields = append(fields, zap.Error(fmt.Errorf("invalid number of arguments")))
		return fields
	}
	for idx := 0; idx < len(keyvals); idx += 2 {
		if key, ok := keyvals[idx].(string); ok {
			switch value := keyvals[idx+1].(type) {
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
			case error:
				fields = append(fields, zap.Error(value))
			default:
				fields = append(fields, zap.Object(key, value))
			}
		}
	}
	return fields
}
