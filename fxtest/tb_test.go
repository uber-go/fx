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

package fxtest

import (
	"bytes"
	"fmt"
	"testing"
)

// Verify that TB always matches testing.T.
var _ TB = (*testing.T)(nil)

type tb struct {
	failures int
	errors   *bytes.Buffer
	logs     *bytes.Buffer
}

func newTB() *tb {
	return &tb{0, &bytes.Buffer{}, &bytes.Buffer{}}
}

func (t *tb) FailNow() {
	t.failures++
}

func (t *tb) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(t.errors, format, args...)
	t.errors.WriteRune('\n')
}

func (t *tb) Logf(format string, args ...interface{}) {
	fmt.Fprintf(t.logs, format, args...)
	t.logs.WriteRune('\n')
}
