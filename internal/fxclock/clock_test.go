// Copyright (c) 2021 Uber Technologies, Inc.
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
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

var _ Clock = clock.Clock(nil)

// Just a basic sanity check that everything is in order.
func TestSystemClock(t *testing.T) {
	t.Parallel()

	clock := System

	t.Run("Now and Since", func(t *testing.T) {
		t.Parallel()

		before := clock.Now()
		assert.GreaterOrEqual(t, clock.Since(before), time.Duration(0))
	})

	t.Run("Sleep", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			clock.Sleep(time.Millisecond)
		})
	})

	t.Run("WithTimeout", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx, cancel := clock.WithTimeout(ctx, time.Second)
		defer cancel()

		_, ok := ctx.Deadline()
		assert.True(t, ok, "must have deadline")
	})
}
