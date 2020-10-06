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
)

//type FxLogger interface {
//	// Do fatal manually.
//	Error(msg string)
//	Info(msg string)
//	WithFields(fields ...LogField) FxLogger
//}

// LogLevel is the level of logging used by
type LogLevel int

const (
	LogLevelInfo = iota
	LogLevelError
)

type LogEntry struct {
	Level LogLevel
	Message string
	Fields []LogField
	Stack string
}

func (l LogEntry) WithStack(stack string) LogEntry {
	l.Stack = stack

	return l
}

func (l LogEntry) Write(logger *Logger) {
	logger.core.Log(l)
}

type LogField struct {
	Key string
	Value interface{}
}

type CoreLogger interface {
	Log(entry LogEntry)
}

var _ CoreLogger = (*coreLogger)(nil)

type coreLogger struct {
	log *zap.Logger
}

func EncodeFields(fields []LogField) []zap.Field {
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

func (c *coreLogger) Log(entry LogEntry) {
	switch entry.Level {
	case LogLevelInfo:
		c.log.Info(entry.Message, EncodeFields(entry.Fields)...)
	case LogLevelError:
		c.log.Error(entry.Message, EncodeFields(entry.Fields)...)
	}
}

// Take printer as argument, maybe.
func DefaultLogger() *Logger {
	log, _ := zap.NewProduction()

	return &Logger{
		core: &coreLogger{
			log: log,
		},
	}
}

type Logger struct{
	core CoreLogger
}

func Info(msg string, fields ...LogField) LogEntry {
	return LogEntry{
		Level: LogLevelInfo,
		Message: msg,
		Fields: fields,
	}
}

func Error(msg string, fields ...LogField) LogEntry {
	return LogEntry{
		Level: LogLevelError,
		Message: msg,
		Fields: fields,
	}

}

//func (*Logger) PrintProvide(x interface{}) {
//
//}
//
//func (*Logger) PrintSupply(x interface{}) {
//
//}

//var _exit = func() { os.Exit(1) }
//
//// Printer is a formatting printer.
//type Printer interface {
//	Printf(string, ...interface{})
//}
//
//// New returns a new Logger backed by the standard library's log package.
//func New() *Logger {
//	return &Logger{log.New(os.Stderr, "", log.LstdFlags)}
//}
//
//// A Logger writes output to standard error.
//type Logger struct {
//	Printer
//}
//
//// Printf logs a formatted Fx line.
//func (l *Logger) Printf(format string, v ...interface{}) {
//	l.Printer.Printf(prepend(format), v...)
//}
//
//// PrintProvide logs a type provided into the dig.Container.
//func (l *Logger) PrintProvide(t interface{}) {
//	for _, rtype := range fxreflect.ReturnTypes(t) {
//		l.Printf("PROVIDE\t%s <= %s", rtype, fxreflect.FuncName(t))
//	}
//}
//
//// PrintSupply logs a type supplied directly into the dig.Container
//// by the given constructor function.
//func (l *Logger) PrintSupply(constructor interface{}) {
//	for _, rtype := range fxreflect.ReturnTypes(constructor) {
//		l.Printf("SUPPLY\t%s", rtype)
//	}
//}
//
//// PrintSignal logs an os.Signal.
//func (l *Logger) PrintSignal(signal os.Signal) {
//	l.Printf(strings.ToUpper(signal.String()))
//}
//
//// Panic logs an Fx line then panics.
////func (l *Logger) Panic(err error) {
////	l.Printer.Printf(prepend(err.Error()))
////	panic(err)
////}
//
//// Fatalf logs an Fx line then fatals.
////func (l *Logger) Fatalf(format string, v ...interface{}) {
////	l.Printer.Printf(prepend(format), v...)
////	_exit()
////}
//
//func prepend(str string) string {
//	return fmt.Sprintf("[Fx] %s", str)
//}
