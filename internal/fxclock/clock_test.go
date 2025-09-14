// Copyright (c) 2024 Uber Technologies, Inc.
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

package fxclock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemClock(t *testing.T) {
	clock := System
	testClock(t, System, clock.Sleep)
}

func TestMockClock(t *testing.T) {
	clock := NewMock()
	testClock(t, clock, clock.Add)
}

func testClock(t *testing.T, clock Clock, advance func(d time.Duration)) {
	now := clock.Now()
	assert.False(t, now.IsZero())

	t.Run("Since", func(t *testing.T) {
		advance(1 * time.Millisecond)
		assert.NotZero(t, clock.Since(now), "time must have advanced")
	})

	t.Run("Sleep", func(t *testing.T) {
		start := clock.Now()

		go func() {
			// For the mock clock, there's a chance that advance will be
			// too fast and the Sleep will block forever, waiting for
			// another advance. The mock clock provides
			// AwaitScheduled to help with this.
			//
			// Since that function is not available on the system clock,
			// we'll use upcasting to check for it.
			if awaiter, ok := clock.(interface{ AwaitScheduled(int) }); ok {
				awaiter.AwaitScheduled(1)
			}

			advance(1 * time.Millisecond)
		}()
		clock.Sleep(1 * time.Millisecond)

		assert.NotZero(t, clock.Since(start), "time must have advanced")
	})

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := clock.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		t.Run("Deadline", func(t *testing.T) {
			dl, ok := ctx.Deadline()
			assert.True(t, ok, "must have a deadline")
			assert.True(t, dl.After(now), "deadline must be in the future")
		})

		advance(1 * time.Millisecond)

		select {
		case <-ctx.Done():
			assert.Error(t, ctx.Err(), "done context must error")
			assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded,
				"context must have exceeded its deadline")

		case <-time.After(10 * time.Millisecond):
			t.Fatal("expected context to be done")
		}
	})

	t.Run("WithTimeout/Value", func(t *testing.T) {
		type contextKey string
		key := contextKey("foo")

		ctx1 := context.WithValue(context.Background(), key, "bar")

		ctx2, cancel := clock.WithTimeout(ctx1, 1*time.Millisecond)
		defer cancel()

		assert.Equal(t, "bar", ctx2.Value(key), "value must be preserved")
	})

	t.Run("WithTimeout/Cancel", func(t *testing.T) {
		ctx, cancel := clock.WithTimeout(context.Background(), 1*time.Millisecond)
		cancel()

		select {
		case <-ctx.Done():
			assert.Error(t, ctx.Err(), "done context must error")
			assert.ErrorIs(t, ctx.Err(), context.Canceled,
				"context must have been canceled")

		case <-time.After(10 * time.Millisecond):
			t.Fatal("expected context to be done")
		}
	})
}

func TestMock_Sleep(t *testing.T) {
	clock := NewMock()

	ch := make(chan struct{})
	go func() {
		clock.Sleep(2 * time.Millisecond)
		close(ch)
	}()

	// We cannot advance time until we're certain
	// that the Sleep call has started waiting.
	// Otherwise, we'll advance that one millisecond,
	// and then the Sleep will start waiting for another Advance,
	// which will never come.
	//
	// AwaitScheduled will block until there is at least one
	// scheduled event.
	clock.AwaitScheduled(1)

	// Advance only one millisecond, the Sleep should not return.
	clock.Add(1 * time.Millisecond)
	select {
	case <-ch:
		t.Fatal("sleep should not have returned")
	case <-time.After(1 * time.Millisecond):
		// ok
	}

	// Avance to the next millisecond, the Sleep should return.
	clock.Add(1 * time.Millisecond)
	select {
	case <-ch:
		// ok
	case <-time.After(10 * time.Millisecond):
		t.Fatal("expected Sleep to return")
	}
}

func TestMock_AddNegative(t *testing.T) {
	clock := NewMock()
	assert.Panics(t, func() { clock.Add(-1) })
}

func TestMock_ManySleepers(t *testing.T) {
	const N = 100

	clock := NewMock()

	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()

			clock.Sleep(1 * time.Millisecond)
		}()
	}

	clock.AwaitScheduled(N)
	clock.Add(1 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		// ok
	case <-time.After(10 * time.Millisecond):
		t.Fatal("expected all sleepers to be done")
	}
}
