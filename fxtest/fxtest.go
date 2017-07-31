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
	"context"

	"go.uber.org/fx"
	"go.uber.org/fx/internal/lifecycle"
)

// TB is a subset of the standard library's testing.TB interface. It's
// satisfied by both *testing.T and *testing.B.
type TB interface {
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	FailNow()
}

type testPrinter struct {
	TB
}

func (p *testPrinter) Printf(format string, args ...interface{}) {
	p.Logf(format, args...)
}

// App is a wrapper around fx.App that provides some testing helpers. By
// default, it uses the provided TB as the application's logging backend.
type App struct {
	*fx.App

	tb TB
}

// New creates a new test application.
func New(tb TB, opts ...fx.Option) *App {
	allOpts := make([]fx.Option, 0, len(opts)+1)
	allOpts = append(allOpts, fx.Logger(&testPrinter{tb}))
	allOpts = append(allOpts, opts...)
	return &App{
		App: fx.New(allOpts...),
		tb:  tb,
	}
}

// RequireStart calls Start, failing the test if an error is encountered.
func (app *App) RequireStart() *App {
	if err := app.Start(context.Background()); err != nil {
		app.tb.Errorf("application didn't start cleanly: %v", err)
		app.tb.FailNow()
	}
	return app
}

// RequireStop calls Stop, failing the test if an error is encountered.
func (app *App) RequireStop() {
	if err := app.Stop(context.Background()); err != nil {
		app.tb.Errorf("application didn't stop cleanly: %v", err)
		app.tb.FailNow()
	}
}

var _ fx.Lifecycle = (*Lifecycle)(nil)

// Lifecycle is a testing spy for fx.Lifecycle. It exposes Start and Stop
// methods (and some test-specific helpers) so that unit tests can exercise
// hooks.
type Lifecycle struct {
	t  TB
	lc *lifecycle.Lifecycle
}

// NewLifecycle creates a new test lifecycle.
func NewLifecycle(t TB) *Lifecycle {
	return &Lifecycle{
		lc: lifecycle.New(nil),
		t:  t,
	}
}

// Start executes all registered OnStart hooks in order, halting at the first
// hook that doesn't succeed.
func (l *Lifecycle) Start(ctx context.Context) error { return l.lc.Start(ctx) }

// RequireStart calls Start with context.Background(), failing the test if an
// error is encountered.
func (l *Lifecycle) RequireStart() *Lifecycle {
	if err := l.Start(context.Background()); err != nil {
		l.t.Errorf("lifecycle didn't start cleanly: %v", err)
		l.t.FailNow()
	}
	return l
}

// Stop calls all OnStop hooks whose OnStart counterpart was called, running
// in reverse order.
//
// If any hook returns an error, execution continues for a best-effort
// cleanup. Any errors encountered are collected into a single error and
// returned.
func (l *Lifecycle) Stop(ctx context.Context) error { return l.lc.Stop(ctx) }

// RequireStop calls Stop with context.Background(), failing the test if an error
// is encountered.
func (l *Lifecycle) RequireStop() {
	if err := l.Stop(context.Background()); err != nil {
		l.t.Errorf("lifecycle didn't stop cleanly: %v", err)
		l.t.FailNow()
	}
}

// Append registers a new Hook.
func (l *Lifecycle) Append(h fx.Hook) {
	l.lc.Append(lifecycle.Hook{
		OnStart: h.OnStart,
		OnStop:  h.OnStop,
	})
}
