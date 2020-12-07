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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level is the level of logging used by Logger.
type Level int

const (
	InfoLevel Level = iota
	ErrorLevel
)

// Entry is an entry to be later serialized into zap message and fields.
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

func F(key string, value interface{}) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

type Field struct {
	Key   string
	Value interface{}
}

type Logger interface {
	Log(entry Entry)
}

var _ Logger = (*zapLogger)(nil)

type zapLogger struct {
	logger *zap.Logger
}

func encodeFields(fields []Field, stack string) []zap.Field {
	var fs []zap.Field
	for _, field := range fields {
		fs = append(fs, zap.Any(field.Key, field.Value))
	}
	if stack != "" {
		fs = append(fs, zap.Stack(stack))
	}

	return fs
}

func (l *zapLogger) Log(entry Entry) {
	switch entry.Level {
	case InfoLevel:
		l.logger.Info(entry.Message, encodeFields(entry.Fields, entry.Stack)...)
	case ErrorLevel:
		l.logger.Error(entry.Message, encodeFields(entry.Fields, entry.Stack)...)
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

// Info creates a logging Info entry.
func Info(msg string, fields ...Field) Entry {
	return Entry{
		Level:   InfoLevel,
		Message: msg,
		Fields:  fields,
	}
}

// Error creates a logging Error entry.
func Error(msg string, fields ...Field) Entry {
	return Entry{
		Level:   ErrorLevel,
		Message: msg,
		Fields:  fields,
	}
}
