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

package fxtest

import (
	"go.uber.org/fx"
	"go.uber.org/fx/internal/lifecycle"
)

// TB is a subset of the standard library's testing.TB interface. It's
// satisfied by both *testing.T and *testing.B.
type TB interface {
	Errorf(string, ...interface{})
	FailNow()
}

// Lifecycle is a testing spy for fx.Lifecycle. It exposes Start and Stop
// methods (and some test-specific helpers) so that unit tests can exercise
// hooks.
type Lifecycle struct {
	*lifecycle.Lifecycle

	t TB
}

// NewLifecycle creates a new test lifecycle.
func NewLifecycle(t TB) *Lifecycle {
	return &Lifecycle{
		Lifecycle: lifecycle.NewLifecycle(nil),
		t:         t,
	}
}

// Start executes all registered OnStart hooks in order, halting at the first
// hook that doesn't succeed.
func (l *Lifecycle) Start() error { return l.Lifecycle.Start() }

// MustStart calls Start, failing the test if an error is encountered.
func (l *Lifecycle) MustStart() {
	if err := l.Start(); err != nil {
		l.t.Errorf("lifecycle didn't start cleanly: %v", err)
		l.t.FailNow()
	}
}

// Stop calls all OnStop hooks whose OnStart counterpart was called, running
// in reverse order.
//
// If any hook returns an error, execution continues for a best-effort
// cleanup. Any errors encountered are collected into a single error and
// returned.
func (l *Lifecycle) Stop() error { return l.Lifecycle.Stop() }

// MustStop calls Stop, failing the test if an error is encountered.
func (l *Lifecycle) MustStop() {
	if err := l.Stop(); err != nil {
		l.t.Errorf("lifecycle didn't stop cleanly: %v", err)
		l.t.FailNow()
	}
}

// Append registers a new Hook.
func (l *Lifecycle) Append(h fx.Hook) {
	l.Lifecycle.Append(lifecycle.Hook{
		OnStart: h.OnStart,
		OnStop:  h.OnStop,
	})
}
