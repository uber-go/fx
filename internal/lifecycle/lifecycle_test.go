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

package lifecycle

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/multierr"

	"github.com/stretchr/testify/assert"
)

func TestLifecycleStart(t *testing.T) {
	t.Run("ExecutesInOrder", func(t *testing.T) {
		l := New(nil)
		count := 0

		l.Append(Hook{
			OnStart: func(context.Context) error {
				count++
				assert.Equal(t, 1, count, "expected this starter to be executed first")
				return nil
			},
		})
		l.Append(Hook{
			OnStart: func(context.Context) error {
				count++
				assert.Equal(t, 2, count, "expected this starter to be executed second")
				return nil
			},
		})

		assert.NoError(t, l.Start(context.Background()))
		assert.Equal(t, 2, count)
	})
	t.Run("ErrHaltsChainAndRollsBack", func(t *testing.T) {
		l := New(nil)
		err := errors.New("a starter error")
		starterCount := 0
		stopperCount := 0

		// this event's starter succeeded, so no matter what the stopper should run
		l.Append(Hook{
			OnStart: func(context.Context) error {
				starterCount++
				return nil
			},
			OnStop: func(context.Context) error {
				stopperCount++
				return nil
			},
		})
		// this event's starter fails, so the stopper shouldnt run
		l.Append(Hook{
			OnStart: func(context.Context) error {
				starterCount++
				return err
			},
			OnStop: func(context.Context) error {
				t.Error("this stopper shouldnt run, since the starter in this event failed")
				return nil
			},
		})
		// this event is last in the chain, so it should never run since the previous failed
		l.Append(Hook{
			OnStart: func(context.Context) error {
				t.Error("this starter should never run, since the previous event failed")
				return nil
			},
			OnStop: func(context.Context) error {
				t.Error("this stopper should never run, since the previous event failed")
				return nil
			},
		})

		assert.Error(t, err, l.Start(context.Background()))
		assert.NoError(t, l.Stop(context.Background()))

		assert.Equal(t, 2, starterCount, "expected the first and second starter to execute")
		assert.Equal(t, 1, stopperCount, "expected the first stopper to execute since the second starter failed")
	})
}

func TestLifecycleStop(t *testing.T) {
	t.Run("DoesNothingWithoutHooks", func(t *testing.T) {
		l := &Lifecycle{logger: fxlog.New()}
		assert.Nil(t, l.Stop(context.Background()), "no lifecycle hooks should have resulted in stop returning nil")
	})

	t.Run("ExecutesInReverseOrder", func(t *testing.T) {
		l := &Lifecycle{logger: fxlog.New()}
		count := 2

		l.Append(Hook{
			OnStop: func(context.Context) error {
				count--
				assert.Equal(t, 0, count, "this stopper was added first, so should execute last")
				return nil
			},
		})
		l.Append(Hook{
			OnStop: func(context.Context) error {
				count--
				assert.Equal(t, 1, count, "this stopper was added last, so should execute first")
				return nil
			},
		})

		assert.NoError(t, l.Start(context.Background()))
		assert.NoError(t, l.Stop(context.Background()))
		assert.Equal(t, 0, count)
	})

	t.Run("ErrDoesntHaltChain", func(t *testing.T) {
		l := New(nil)
		count := 0

		l.Append(Hook{
			OnStop: func(context.Context) error {
				count++
				return nil
			},
		})
		err := errors.New("some stop error")
		l.Append(Hook{
			OnStop: func(context.Context) error {
				count++
				return err
			},
		})

		assert.NoError(t, l.Start(context.Background()))
		assert.Equal(t, err, l.Stop(context.Background()))
		assert.Equal(t, 2, count)
	})
	t.Run("GathersAllErrs", func(t *testing.T) {
		l := New(nil)

		err := errors.New("some stop error")
		err2 := errors.New("some other stop error")

		l.Append(Hook{
			OnStop: func(context.Context) error {
				return err2
			},
		})
		l.Append(Hook{
			OnStop: func(context.Context) error {
				return err
			},
		})

		assert.NoError(t, l.Start(context.Background()))
		assert.Equal(t, multierr.Combine(err, err2), l.Stop(context.Background()))
	})
	t.Run("AllowEmptyHooks", func(t *testing.T) {
		l := New(nil)
		l.Append(Hook{})
		l.Append(Hook{})

		assert.NoError(t, l.Start(context.Background()))
		assert.NoError(t, l.Stop(context.Background()))
	})
}
