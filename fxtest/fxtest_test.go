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
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

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

func (t *tb) Reset() {
	t.failures = 0
	t.errors.Reset()
	t.logs.Reset()
}

func TestApp(t *testing.T) {
	spy := newTB()

	t.Run("Success", func(t *testing.T) {
		spy.Reset()

		New(spy).RequireStart().RequireStop()

		assert.Zero(t, spy.failures, "App didn't start and stop cleanly.")
		assert.Contains(t, spy.logs.String(), "RUNNING", "Expected to write logs to TB.")
	})

	t.Run("StartFailure", func(t *testing.T) {
		spy.Reset()

		New(
			spy,
			fx.Invoke(func() error { return errors.New("fail") }),
		).RequireStart()

		assert.Equal(t, 1, spy.failures, "Expected app to error on start.")
		assert.Contains(t, spy.errors.String(), "didn't start cleanly", "Expected to write errors to TB.")
	})

	t.Run("StopFailure", func(t *testing.T) {
		spy.Reset()

		construct := func(lc fx.Lifecycle) struct{} {
			lc.Append(fx.Hook{OnStop: func(context.Context) error { return errors.New("fail") }})
			return struct{}{}
		}
		New(
			spy,
			fx.Provide(construct),
			fx.Invoke(func(struct{}) {}),
		).RequireStart().RequireStop()

		assert.Equal(t, 1, spy.failures, "Expected Stop to fail.")
		assert.Contains(t, spy.errors.String(), "didn't stop cleanly", "Expected to write errors to TB.")
	})
}

func TestLifecycle(t *testing.T) {
	spy := newTB()

	t.Run("Success", func(t *testing.T) {
		spy.Reset()

		n := 0
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error { n++; return nil },
			OnStop:  func(context.Context) error { n++; return nil },
		})
		lc.RequireStart().RequireStop()

		assert.Zero(t, spy.failures, "Lifecycle start/stop failed.")
		assert.Equal(t, 2, n, "Didn't run start and stop hooks.")
	})

	t.Run("StartFailure", func(t *testing.T) {
		spy.Reset()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStart: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle start to fail.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to succeed.")
	})

	t.Run("StopFailure", func(t *testing.T) {
		spy.Reset()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStop: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 0, spy.failures, "Expected lifecycle start to succeed.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to fail.")
	})
}
