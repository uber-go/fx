// Copyright (c) 2018 Uber Technologies, Inc.
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

package fx

import (
	"context"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShutdown(t *testing.T) {
	t.Run("ShutsDownApp", func(t *testing.T) {
		var started, stopped bool
		app := New(Invoke(func(l Lifecycle, s Shutdowner) {
			l.Append(Hook{
				OnStart: func(ctx context.Context) error {
					started = true
					return nil
				},
				OnStop: func(ctx context.Context) error {
					stopped = true
					return nil
				},
			})
			l.Append(Hook{
				OnStart: func(ctx context.Context) error { return s.Shutdown() },
			})
		}))

		app.Run()
		assert.True(t, started, "app wasn't started")
		assert.True(t, stopped, "app wasn't stopped")
	})

	t.Run("BroadcastsToMultipleChannels", func(t *testing.T) {
		var started, stopped bool
		app := New(Invoke(func(l Lifecycle, s Shutdowner) {
			l.Append(Hook{
				OnStart: func(ctx context.Context) error {
					started = true
					return nil
				},
				OnStop: func(ctx context.Context) error {
					stopped = true
					return nil
				},
			})
		}))

		done1, done2 := app.Done(), app.Done()
		startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
		defer cancel()
		assert.NoError(t, app.Start(startCtx), "error in app start")

		assert.NoError(t, app.shutdowner().Shutdown())

		assert.Equal(t, syscall.SIGTERM, <-done1)
		assert.Equal(t, syscall.SIGTERM, <-done2)

		stopCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
		defer cancel()
		assert.NoError(t, app.Stop(stopCtx), "error in app stop")
		assert.True(t, started, "app wasn't started")
		assert.True(t, stopped, "app wasn't stopped")
	})

	t.Run("ErrorOnUnsentSignal", func(t *testing.T) {
		var started, stopped bool
		app := New(Invoke(func(l Lifecycle, s Shutdowner) {
			l.Append(Hook{
				OnStart: func(ctx context.Context) error {
					started = true
					return nil
				},
				OnStop: func(ctx context.Context) error {
					stopped = true
					return nil
				},
			})
		}))

		done := app.Done()
		s := app.shutdowner().(*shutdowner)
		s.broadcastSignal(syscall.SIGINT)

		startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
		assert.NoError(t, app.Start(startCtx), "error in app start")
		defer cancel()

		err := s.Shutdown()
		assert.Error(t, err, "no error returned from unsent signal")
		assert.Equal(t, syscall.SIGINT, <-done)

		stopCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
		defer cancel()
		assert.NoError(t, app.Stop(stopCtx), "error in app stop")
		assert.True(t, started, "app wasn't started")
		assert.True(t, stopped, "app wasn't stopped")
	})
}
