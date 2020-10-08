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
	"os"
)

// LogLevel is the level of logging used by Core.
type LogLevel int

const (
	InfoLevel = iota
	ErrorLevel
)

// Entry is an entry
type Entry struct {
	Level LogLevel
	Message string
	Fields []Field
	Stack string
}

func (e Entry) WithStack(stack string) Entry {
	e.Stack = stack

	return e
}

func (e Entry) Write(logger *Logger) {
	logger.Core.Log(e)
}

type Field struct {
	Key string
	Value interface{}
}

type Core interface {
	Log(entry Entry)
}

var _ Core = (*LogCore)(nil)

type LogCore struct {
	Zlog *zap.Logger
}

func EncodeFields(fields []Field) []zap.Field {
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

func (c *LogCore) Log(entry Entry) {
	switch entry.Level {
	case InfoLevel:
		c.Zlog.Info(entry.Message, EncodeFields(entry.Fields)...)
	case ErrorLevel:
		c.Zlog.Error(entry.Message, EncodeFields(entry.Fields)...)
	}
}

// Take printer as argument, maybe.
func DefaultLogger() *Logger {
	zcore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(os.Stderr),
		zap.NewAtomicLevel(),
		)
	log := zap.New(zcore)

	return &Logger{
		Core: &LogCore{
			Zlog: log,
		},
	}
}

type Logger struct{
	Core Core
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
