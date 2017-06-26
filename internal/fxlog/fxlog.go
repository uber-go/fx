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

package fxlog

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"go.uber.org/fx/internal/fxreflect"
)

var _exit = func() { os.Exit(1) }

type printer interface {
	Printf(string, ...interface{})
}

// New returns a new Logger.
func New() *Logger {
	return &Logger{log.New(os.Stderr, "", log.LstdFlags)}
}

// A Logger writes output to standard error.
type Logger struct {
	std printer
}

// Println logs a single Fx line.
func (l *Logger) Println(str string) {
	l.std.Printf(prepend(str))
}

// Printf logs a formatted Fx line.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.std.Printf(prepend(format), v...)
}

// PrintProvide logs a type provided into the dig.Container.
func (l *Logger) PrintProvide(t interface{}) {
	if reflect.TypeOf(t).Kind() != reflect.Func {
		// Invalid provide, will be logged as an error.
		return
	}
	for _, rtype := range fxreflect.ReturnTypes(t) {
		l.Printf("PROVIDE\t%s <= %s", rtype, fxreflect.FuncName(t))
	}
}

// PrintSignal logs an os.Signal.
func (l *Logger) PrintSignal(signal os.Signal) {
	l.Println(strings.ToUpper(signal.String()))
}

// Panic logs an Fx line then panics.
func (l *Logger) Panic(err error) {
	l.std.Printf(prepend(err.Error()))
	panic(err)
}

// Fatalf logs an Fx line then fatals.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.std.Printf(prepend(format), v...)
	_exit()
}

func prepend(str string) string {
	return fmt.Sprintf("[Fx] %s", str)
}
