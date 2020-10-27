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
	"go.uber.org/fx/internal/fxreflect"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _exit = func() { os.Exit(1) }

// Level is the level of logging used by Logger.
type Level int

const (
	InfoLevel = iota
	ErrorLevel
)

// Entry is an entry
type Entry struct {
	Level   Level
	Message string
	Fields  []Field
	Stack   string
}

func (e Entry) WithStack(stack string) Entry {
	e.Stack = stack

	return e
}

func (e Entry) Write(logger Logger) {
	logger.Log(e)
}

type Field struct {
	Key   string
	Value interface{}
}

type Logger interface {
	Log(entry Entry)
	PrintProvide(interface{})
	PrintSupply(interface{})
}

var _ Logger = (*zapLogger)(nil)

type zapLogger struct {
	logger *zap.Logger
}

func encodeFields(fields []Field) []zap.Field {
	var fs []zap.Field
	for _, field := range fields {
		fs = append(fs, zap.Any(field.Key, field.Value))
	}

	return fs
}

func (l *zapLogger) Log(entry Entry) {
	switch entry.Level {
	case InfoLevel:
		l.logger.Info(entry.Message, encodeFields(entry.Fields)...)
	case ErrorLevel:
		l.logger.Error(entry.Message, encodeFields(entry.Fields)...)
	}
}

func (l *zapLogger) PrintProvide(t interface{}) {
	for _, rtype := range fxreflect.ReturnTypes(t) {
		Info("providing",
			Field{Key: "return value", Value: rtype},
			Field{Key: "constructor", Value: fxreflect.FuncName(t)},
		).Write(l)
	}
}

func (l *zapLogger) PrintSupply(t interface{}) {
	for _, rtype := range fxreflect.ReturnTypes(t) {
		Info("supplying",
			Field{Key: "constructor", Value: rtype},
		).Write(l)
	}
}

// DefaultLogger constructs a Logger out of io.Writer.
func DefaultLogger(ws zapcore.WriteSyncer) Logger {
	zcore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(ws),
		zap.NewAtomicLevel(),
	)
	log := zap.New(zcore)

	return &zapLogger{
		logger: log,
	}
}

func Info(msg string, fields ...Field) Entry {
	return Entry{
		Level:   InfoLevel,
		Message: msg,
		Fields:  fields,
	}
}

func Error(msg string, fields ...Field) Entry {
	return Entry{
		Level:   ErrorLevel,
		Message: msg,
		Fields:  fields,
	}

}
