// Copyright (c) 2019 Uber Technologies, Inc.
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
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxclock"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/internal/testutil"
	"go.uber.org/goleak"
	"go.uber.org/multierr"
)

func testLogger(t *testing.T) fxevent.Logger {
	return fxlog.DefaultLogger(testutil.WriteSyncer{T: t})
}

func TestLifecycleStart(t *testing.T) {
	t.Parallel()

	t.Run("ExecutesInOrder", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
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
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
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

	t.Run("DoNotRunStartHooksWithExpiredCtx", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		l.Append(Hook{
			OnStart: func(context.Context) error {
				assert.Fail(t, "this hook should not run")
				return nil
			},
			OnStop: func(context.Context) error {
				assert.Fail(t, "this hook should not run")
				return nil
			},
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := l.Start(ctx)
		require.Error(t, err)
		// Note: Stop does not return an error here because no hooks
		// have been started, so we don't end up any of the corresponding
		// stop hooks.
		require.NoError(t, l.Stop(ctx))
	})

	t.Run("StartWhileStartedErrors", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		assert.NoError(t, l.Start(context.Background()))
		err := l.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "attempted to start lifecycle when in state: started")
		assert.NoError(t, l.Stop(context.Background()))
		assert.NoError(t, l.Start(context.Background()))
	})
}

func TestLifecycleStop(t *testing.T) {
	t.Parallel()

	t.Run("DoesNothingWithoutHooks", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		l.Start(context.Background())
		assert.Nil(t, l.Stop(context.Background()), "no lifecycle hooks should have resulted in stop returning nil")
	})

	t.Run("DoesNothingWhenNotStarted", func(t *testing.T) {
		t.Parallel()

		hook := Hook{
			OnStop: func(context.Context) error {
				assert.Fail(t, "OnStop should not be called if lifecycle was never started")
				return nil
			},
		}
		l := New(testLogger(t), fxclock.System)
		l.Append(hook)
		l.Stop(context.Background())
	})

	t.Run("ExecutesInReverseOrder", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
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
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
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
		t.Parallel()

		l := New(testLogger(t), fxclock.System)

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
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		l.Append(Hook{})
		l.Append(Hook{})

		assert.NoError(t, l.Start(context.Background()))
		assert.NoError(t, l.Stop(context.Background()))
	})

	t.Run("DoesNothingIfStartFailed", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		err := errors.New("some start error")

		l.Append(Hook{
			OnStart: func(context.Context) error {
				return err
			},
			OnStop: func(context.Context) error {
				assert.Fail(t, "OnStop should not be called if start failed")
				return nil
			},
		})

		assert.Equal(t, err, l.Start(context.Background()))
		l.Stop(context.Background())
	})

	t.Run("DoNotRunStopHooksWithExpiredCtx", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		l.Append(Hook{
			OnStart: func(context.Context) error {
				return nil
			},
			OnStop: func(context.Context) error {
				assert.Fail(t, "this hook should not run")
				return nil
			},
		})
		ctx, cancel := context.WithCancel(context.Background())
		err := l.Start(ctx)
		require.NoError(t, err)
		cancel()
		require.Error(t, l.Stop(ctx))
	})

	t.Run("nil ctx", func(t *testing.T) {
		t.Parallel()

		l := New(testLogger(t), fxclock.System)
		l.Append(Hook{
			OnStart: func(context.Context) error {
				assert.Fail(t, "this hook should not run")
				return nil
			},
			OnStop: func(context.Context) error {
				assert.Fail(t, "this hook should not run")
				return nil
			},
		})
		//lint:ignore SA1012 this test specifically tests for the lint failure
		err := l.Start(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "called OnStart with nil context")

		//lint:ignore SA1012 this test specifically tests for the lint failure
		err = l.Stop(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "called OnStop with nil context")

	})
}

func TestHookRecordsFormat(t *testing.T) {
	t.Parallel()

	t.Run("SortRecords", func(t *testing.T) {
		t.Parallel()

		t1, err := time.ParseDuration("10ms")
		require.NoError(t, err)
		t2, err := time.ParseDuration("20ms")
		require.NoError(t, err)

		f := fxreflect.Frame{
			Function: "someFunc",
			File:     "somefunc.go",
			Line:     1,
		}

		r := HookRecords{
			HookRecord{
				CallerFrame: f,
				Func: func(context.Context) error {
					return nil
				},
				Runtime: t1,
			},
			HookRecord{
				CallerFrame: f,
				Func: func(context.Context) error {
					return nil
				},
				Runtime: t2,
			},
		}

		sort.Sort(r)
		for _, format := range []string{"%v", "%+v", "%s"} {
			s := fmt.Sprintf(format, r)
			hook1Idx := strings.Index(s, "TestHookRecordsFormat.func1.1()")
			hook2Idx := strings.Index(s, "TestHookRecordsFormat.func1.2()")
			assert.Greater(t, hook1Idx, hook2Idx, "second hook must appear first in the formatted string")
			assert.Contains(t, s, "somefunc.go:1", "file name and line should be reported")
		}
	})
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
