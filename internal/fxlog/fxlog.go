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

// Logger logs human-readable output
type Logger interface {
	Println(string)
	Printf(string, ...interface{})
	Panic(error)
	Fatalf(string, ...interface{})
}

// New returns a new StdLogger
func New() *StdLogger {
	return &StdLogger{
		l: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// StdLogger outputs logs to stderr
type StdLogger struct {
	l *log.Logger
}

// Println logs a single Fx line
func (s StdLogger) Println(str string) {
	s.l.Println(prepend(str))
}

// Printf logs a formatted Fx line
func (s StdLogger) Printf(format string, v ...interface{}) {
	s.l.Printf(prepend(format), v...)
}

// Panic logs an Fx line then panics
func (s StdLogger) Panic(err error) {
	s.l.Panic(prepend(err.Error()))
}

// Fatalf logs an Fx line then fatals
func (s StdLogger) Fatalf(format string, v ...interface{}) {
	s.l.Fatalf(prepend(format), v...)
}

func prepend(str string) string {
	return fmt.Sprintf("[Fx] %s", str)
}

// PrintProvide logs a type provided into the dig.Container
func PrintProvide(l Logger, t interface{}) {
	if reflect.TypeOf(t).Kind() == reflect.Func {
		for _, rtype := range fxreflect.ReturnTypes(t) {
			l.Printf("PROVIDE\t%s <= %s", rtype, fxreflect.FuncName(t))
		}
	} else {
		l.Printf("PROVIDE\t%s", reflect.TypeOf(t).String())
	}
}

// PrintSignal logs an os.Signal
func PrintSignal(l Logger, signal os.Signal) {
	fmt.Println("")
	l.Println(strings.ToUpper(signal.String()))
}
