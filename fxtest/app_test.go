// Copyright (c) 2019-2021 Uber Technologies, Inc.
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
)

func TestApp(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		spy := newTB()

		New(spy).RequireStart().RequireStop()

		assert.Zero(t, spy.failures, "App didn't start and stop cleanly.")
		assert.Contains(t, spy.logs.String(), "[Fx] RUNNING", "Expected to write logs to TB.")
	})

	t.Run("NewFailure", func(t *testing.T) {
		spy := newTB()

		New(
			spy,

			// Missing string dependency in container.
			fx.Invoke(func(string) {}),
		)

		assert.Equal(t, 1, spy.failures, "Expected app to error on New.")
		assert.Contains(t, spy.errors.String(), "New failed", "Expected to write error to TB")
	})

	t.Run("StartError", func(t *testing.T) {
		spy := newTB()

		New(
			spy,

			fx.Invoke(func(lc fx.Lifecycle) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error { return errors.New("fail") },
				})
			}),
		).RequireStart()

		assert.Equal(t, 1, spy.failures, "Expected app to error on start.")
		assert.Contains(t, spy.errors.String(), "didn't start cleanly", "Expected to write errors to TB.")
	})

	t.Run("StopFailure", func(t *testing.T) {
		spy := newTB()

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
