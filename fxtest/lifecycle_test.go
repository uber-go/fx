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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/goleak"
)

func TestLifecycle(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	t.Run("Success", func(t *testing.T) {
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

	t.Run("StartFailure", func(t *testing.T) {
		spy := newTB()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStart: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle start to fail.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to succeed.")
	})

	t.Run("StopFailure", func(t *testing.T) {
		spy := newTB()
		lc := NewLifecycle(spy)
		lc.Append(fx.Hook{OnStop: func(context.Context) error { return errors.New("fail") }})

		lc.RequireStart()
		assert.Equal(t, 0, spy.failures, "Expected lifecycle start to succeed.")

		lc.RequireStop()
		assert.Equal(t, 1, spy.failures, "Expected lifecycle stop to fail.")
	})

	t.Run("RequireLeakDetection", func(t *testing.T) {
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
