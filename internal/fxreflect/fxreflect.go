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

package fxreflect

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// ReturnTypes takes a func and returns a slice of string'd types
// TODO instead of duplicating the dig's reflect logic, trying to
// determine which types actually made it into the container, have
// dig return a Result struct which could contain the ProvidedTypes
func ReturnTypes(t interface{}) []string {
	rtypes := []string{}
	fn := reflect.ValueOf(t).Type()

	for i := 0; i < fn.NumOut(); i++ {
		if !isErr(fn.Out(i)) {
			rtypes = append(rtypes, fn.Out(i).String())
		}
	}

	return rtypes
}

// Caller returns the formatted calling func name
func Caller() string {
	// Ascend at most 8 frames looking for a caller outside fx.
	pcs := make([]uintptr, 8)

	// Don't include this frame.
	n := runtime.Callers(1, pcs)
	if n == 0 {
		return "n/a"
	}

	frames := runtime.CallersFrames(pcs)
	for f, more := frames.Next(); more; f, more = frames.Next() {
		if shouldIgnoreFrame(f) {
			continue
		}
		return f.Function
	}
	return "n/a"
}

// FuncName returns a funcs formatted name
func FuncName(fn interface{}) string {
	fnV := reflect.ValueOf(fn)
	if fnV.Kind() != reflect.Func {
		return "n/a"
	}

	fnName := runtime.FuncForPC(fnV.Pointer()).Name()
	return fmt.Sprintf("%s()", fnName)
}

func isErr(t reflect.Type) bool {
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	return t.Implements(errInterface)
}

// Ascend the call stack until we leave the Fx production code. This allows us
// to avoid hard-coding a frame skip, which makes this code work well even
// when it's wrapped.
func shouldIgnoreFrame(f runtime.Frame) bool {
	if strings.Contains(f.File, "_test.go") {
		return false
	}
	if strings.Contains(f.File, "go.uber.org/fx") {
		return true
	}
	return false
}
