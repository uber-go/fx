// Copyright (c) 2022 Uber Technologies, Inc.
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

package test

import (
	"fmt"
	"runtime"
	"sync"
)

// FakeReport is the result of an execution with a fake T.
type FakeReport struct {
	Failed  bool
	Fatally bool
	Errors  []string
}

// WithFake runs the given function with a fake T.
// Failures reported to the fake will not affect the current test.
// When the given function finishes running,
// WithFake produces a report of the test run.
func WithFake(t T, f func(T)) FakeReport {
	fake := fakeT{T: t}

	// t.FailNow calls runtime.Goexit to kill the current goroutine.
	// To guard against that, we'll run the given function
	// in a separate goroutine and if the user calls t.FailNow,
	// we'll record it and continue execution here.
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(&fake)
	}()
	<-done

	return FakeReport{
		Failed:  fake.failed,
		Fatally: fake.fatal,
		Errors:  fake.errors,
	}
}

// fakeT is a fake implementation of a T.
type fakeT struct {
	T
	mu     sync.RWMutex
	fatal  bool
	failed bool
	errors []string
}

var _ T = (*fakeT)(nil)

func (t *fakeT) Errorf(msg string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.failed = true
	t.errors = append(t.errors, fmt.Sprintf(msg, args...))
}

func (t *fakeT) FailNow() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.failed = true
	t.fatal = true
	runtime.Goexit()
}
