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
	"io"
)

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
	Key string
	Value interface{}
}

type Logger interface {
	Log(entry Entry)
}

var _ Logger = (*zapCore)(nil)

type zapCore struct {
	zapLogger *zap.Logger
}

func encodeFields(fields []Field) []zap.Field {
	var fs []zap.Field
	for _, field := range fields {
		f := zap.Field{
			Key: field.Key,
			Interface: field.Value,
		}
		fs = append(fs, f)
	}

	return fs
}

func (c *zapCore) Log(entry Entry) {
	switch entry.Level {
	case InfoLevel:
		c.zapLogger.Info(entry.Message, encodeFields(entry.Fields)...)
	case ErrorLevel:
		c.zapLogger.Error(entry.Message, encodeFields(entry.Fields)...)
	}
}

// DefaultLogger constructs a Logger out of io.Writer.
func DefaultLogger(w io.Writer) Logger {
	ws := zapcore.AddSync(w)
	zcore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(ws),
		zap.NewAtomicLevel(),
	)
	log := zap.New(zcore)

	return &zapCore{
			zapLogger: log,
		}
}

//type Logger struct{
//	Logger Logger
//}

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
