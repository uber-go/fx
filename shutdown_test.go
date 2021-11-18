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

package fx_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("BroadcastsToMultipleChannels", func(t *testing.T) {
		t.Parallel()

		var s fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&s),
		)

		done1, done2 := app.Done(), app.Done()
		defer app.RequireStart().RequireStop()

		assert.NoError(t, s.Shutdown(), "error in app shutdown")
		assert.NotNil(t, <-done1, "done channel 1 did not receive signal")
		assert.NotNil(t, <-done2, "done channel 2 did not receive signal")
	})

	t.Run("ErrorOnUnsentSignal", func(t *testing.T) {
		t.Parallel()

		var s fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&s),
		)

		done := app.Done()
		defer app.RequireStart().RequireStop()
		assert.NoError(t, s.Shutdown(), "error returned from first shutdown call")

		assert.EqualError(t, s.Shutdown(), "failed to send terminated signal to 1 out of 1 channels",
			"unexpected error returned when shutdown is called with a blocked channel")
		assert.NotNil(t, <-done, "done channel did not receive signal")
	})

	t.Run("shutdown app before calling Done()", func(t *testing.T) {
		t.Parallel()

		var s fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&s),
		)

		require.NoError(t, app.Start(context.Background()), "error starting app")
		assert.NoError(t, s.Shutdown(), "error in app shutdown")
		done1, done2 := app.Done(), app.Done()
		defer app.Stop(context.Background())
		// Receiving on done1 and done2 will deadlock in the event that app.Done()
		// doesn't work as expected.
		assert.NotNil(t, <-done1, "done channel 1 did not receive signal")
		assert.NotNil(t, <-done2, "done channel 2 did not receive signal")
	})
}

func TestDataRace(t *testing.T) {
	t.Parallel()

	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Populate(&s),
	)
	require.NoError(t, app.Start(context.Background()), "error starting app")

	const N = 50
	ready := make(chan struct{}) // used to orchestrate goroutines for Done() and ShutdownOption()
	var wg sync.WaitGroup        // tracks and waits for all goroutines

	// Spawn N goroutines, each of which call app.Done() and assert
	// the signal received.
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-ready
			done := app.Done()
			assert.NotNil(t, <-done, "done channel %v did not receive signal", i)
		}()
	}

	// call Shutdown()
	wg.Add(1)
	go func() {
		<-ready
		defer wg.Done()
		assert.NoError(t, s.Shutdown(), "error in app shutdown")
	}()

	close(ready)
	wg.Wait()
}
