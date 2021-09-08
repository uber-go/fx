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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/goleak"
)

func TestLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		spy := newTB()

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

	t.Run("StartError", func(t *testing.T) {
		t.Parallel()

		spy := newTB()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStart: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle start to fail.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to succeed.")
	})

	t.Run("StopFailure", func(t *testing.T) {
		t.Parallel()

		spy := newTB()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStop: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 0, spy.failures, "Expected lifecycle start to succeed.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to fail.")
	})

	t.Run("RequireLeakDetection", func(t *testing.T) {
		t.Parallel()

		spy := newTB()
		lc := NewLifecycle(spy)

		stop := make(chan struct{})
		stopped := make(chan struct{})

		onStart := func(ctx context.Context) error {
			go func() {
				<-stop
				close(stopped)
			}()
			return ctx.Err()
		}

		onStop := func(ctx context.Context) error {
			close(stop)
			<-stopped
			return ctx.Err()
		}

		lc.Append(fx.Hook{
			OnStart: onStart,
			OnStop:  onStop,
		})

		lc.RequireStart()
		assert.Equal(t, 0, spy.failures, "Expected lifecycle start to succeed.")

		lc.RequireStop()
		assert.Equal(t, 0, spy.failures, "Expected lifecycle to stop.")
	})
}

func TestLifecycle_OptionalT(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		lc := NewLifecycle(nil)

		var started, stopped bool
		defer func() {
			assert.True(t, started, "not started")
			assert.True(t, stopped, "not stopped")
		}()
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				assert.False(t, started, "started twice")
				started = true
				return nil
			},
			OnStop: func(context.Context) error {
				assert.True(t, started, "not yet started")
				assert.False(t, stopped, "stopped twice")
				stopped = true
				return nil
			},
		})

		lc.RequireStart().RequireStop()
	})

	t.Run("start error", func(t *testing.T) {
		t.Parallel()

		lc := NewLifecycle(nil)
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				return errors.New("great sadness")
			},
		})

		var pval interface{}
		func() {
			defer func() { pval = recover() }()
			lc.RequireStart()
		}()
		require.NotNil(t, pval, "must panic in case of failure")
		assert.Contains(t, fmt.Sprint(pval), "great sadness")
	})
}

func TestPanicT(t *testing.T) {
	t.Parallel()

	t.Run("Logf", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		pt := panicT{W: &buff}
		pt.Logf("hello: %v", "world")

		assert.Equal(t, "hello: world\n", buff.String())
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		pt := panicT{W: &buff}
		pt.Errorf("hello: %v", "world")

		assert.Equal(t, "hello: world\n", buff.String())

		// Functionally there's no difference between Logf and Errorf
		// unless FailNow is called.
		t.Run("FailNow", func(t *testing.T) {
			t.Parallel()

			var pval interface{}
			func() {
				defer func() { pval = recover() }()
				pt.FailNow()
			}()
			assert.Equal(t, "hello: world", pval)
		})
	})

	// FailNow without calling Errorf will use the fixed message.
	t.Run("FailNow", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		pt := panicT{W: &buff}
		pt.Logf("hello: %v", "world")

		var pval interface{}
		func() {
			defer func() { pval = recover() }()
			pt.FailNow()
		}()
		assert.Equal(t, "test lifecycle failed", pval)
	})
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
