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

// Println logs a single Fx line
func Println(str string) {
	log.Println(prepend(str))
}

// Printf logs a formatted Fx line
func Printf(format string, v ...interface{}) {
	log.Printf(prepend(format), v...)
}

// Panic logs an Fx line then panics
func Panic(err error) {
	log.Panic(prepend(err.Error()))
}

// Fatalf logs an Fx line then fatals
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(prepend(format), v...)
}

func prepend(str string) string {
	return fmt.Sprintf("[Fx] %s", str)
}

// PrintProvide logs a type provided into the dig.Container
func PrintProvide(t interface{}) {
	if reflect.TypeOf(t).Kind() == reflect.Func {
		for _, rtype := range fxreflect.ReturnTypes(t) {
			Printf("PROVIDE\t%s <= %s", rtype, fxreflect.FuncName(t))
		}
	} else {
		Printf("PROVIDE\t%s", reflect.TypeOf(t).String())
	}
}

// PrintSignal logs an os.Signal
func PrintSignal(signal os.Signal) {
	fmt.Println("")
	Println(strings.ToUpper(signal.String()))
}
