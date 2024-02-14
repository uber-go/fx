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
	"sort"
	"sync"
	"time"
)

// Clock defines how Fx accesses time.
// We keep the interface pretty minimal.
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
	Sleep(time.Duration)
	WithTimeout(context.Context, time.Duration) (context.Context, context.CancelFunc)
}

// System is the default implementation of Clock based on real time.
var System Clock = systemClock{}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (systemClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (systemClock) WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

// Mock adapted from
// https://github.com/uber-go/zap/blob/7db06bc9b095571d3dc3d4eebdfbe4dd9bd20405/internal/ztest/clock.go.

// Mock is a fake source of time.
// It implements standard time operations,
// but allows the user to control the passage of time.
//
// Use the [Add] method to progress time.
type Mock struct {
	mu  sync.RWMutex
	now time.Time

	// The MockClock works by maintaining a list of waiters.
	// Each waiter knows the time at which it should be resolved.
	// When the clock advances, all waiters that are in range are resolved
	// in chronological order.
	waiters     []waiter
	waiterAdded *sync.Cond
}

var _ Clock = (*Mock)(nil)

// NewMock builds a new mock clock
// using the current actual time as the initial time.
func NewMock() *Mock {
	m := &Mock{now: time.Now()}
	m.waiterAdded = sync.NewCond(&m.mu)
	return m
}

// Now reports the current time.
func (c *Mock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

// Since reports the time elapsed since t.
// This is short for Now().Sub(t).
func (c *Mock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

// Sleep pauses the current goroutine for the given duration.
//
// With the mock clock, this will freeze
// until the clock is advanced with [Add] past the deadline.
func (c *Mock) Sleep(d time.Duration) {
	ch := make(chan struct{})
	c.runAt(c.Now().Add(d), func() { close(ch) })
	<-ch
}

// WithTimeout returns a new context with a deadline of now + d.
//
// When the deadline is passed, the returned context's Done channel is closed
// and the context's Err method returns context.DeadlineExceeded.
// If the cancel function is called before the deadline is passed,
// the context's Err method returns context.Canceled.
func (c *Mock) WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	// Unfortunately, we can't use context.WithCancelCause here.
	// Per its documentation (and verified by trying it):
	//
	//   ctx, cancel := context.WithCancelCause(parent)
	//   cancel(myError)
	//   ctx.Err() // returns context.Canceled
	//   context.Cause(ctx) // returns myError
	//
	// So it won't do for our purposes.
	deadline := c.Now().Add(d)
	inner, cancelInner := context.WithCancel(ctx)
	dctx := &deadlineCtx{
		inner:       inner,
		cancelInner: cancelInner,
		done:        make(chan struct{}),
		deadline:    deadline,
	}
	ctx = dctx

	c.runAt(deadline, func() {
		dctx.cancel(context.DeadlineExceeded)
	})
	return ctx, func() { dctx.cancel(context.Canceled) }
}

type deadlineCtx struct {
	inner       context.Context
	cancelInner func()

	done     chan struct{}
	deadline time.Time

	mu  sync.Mutex // guards err; the rest is immutable
	err error
}

var _ context.Context = (*deadlineCtx)(nil)

func (c *deadlineCtx) Deadline() (deadline time.Time, ok bool) { return c.deadline, true }
func (c *deadlineCtx) Done() <-chan struct{}                   { return c.done }
func (c *deadlineCtx) Value(key any) any                       { return c.inner.Value(key) }

func (c *deadlineCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *deadlineCtx) cancel(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
		close(c.done)
		c.cancelInner()
	}
	c.mu.Unlock()
}

// runAt schedules the given function to be run at the given time.
// The function runs without a lock held, so it may schedule more work.
func (c *Mock) runAt(t time.Time, fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.waiters = append(c.waiters, waiter{until: t, fn: fn})
	c.waiterAdded.Broadcast()
}

// AwaitScheduled blocks until there are at least N
// operations scheduled for the future.
func (c *Mock) AwaitScheduled(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Note: waiterAdded is associated with c.mu,
	// the same lock we're holding here.
	//
	// When we call Wait(), it'll release the lock
	// and block until signaled by runAt,
	// at which point it'll reacquire the lock
	// (waiting until runAt has released it).
	for len(c.waiters) < n {
		c.waiterAdded.Wait()
	}
}

type waiter struct {
	until time.Time
	fn    func()
}

// Add progresses time by the given duration.
// Other operations waiting for the time to advance
// will be resolved if they are within range.
//
// Side effects of operations waiting for the time to advance
// will take effect on a best-effort basis.
// Avoid racing with operations that have side effects.
//
// Panics if the duration is negative.
func (c *Mock) Add(d time.Duration) {
	if d < 0 {
		panic("cannot add negative duration")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	sort.Slice(c.waiters, func(i, j int) bool {
		return c.waiters[i].until.Before(c.waiters[j].until)
	})

	newTime := c.now.Add(d)
	// newTime won't be recorded until the end of this method.
	// This ensures that any waiters that are resolved
	// are resolved at the time they were expecting.

	for len(c.waiters) > 0 {
		w := c.waiters[0]
		if w.until.After(newTime) {
			break
		}
		c.waiters[0] = waiter{} // avoid memory leak
		c.waiters = c.waiters[1:]

		// The waiter is within range.
		// Travel to the time of the waiter and resolve it.
		c.now = w.until

		// The waiter may schedule more work
		// so we must release the lock.
		c.mu.Unlock()
		w.fn()
		// Sleeping here is necessary to let the side effects of waiters
		// take effect before we continue.
		time.Sleep(1 * time.Millisecond)
		c.mu.Lock()
	}

	c.now = newTime
}
