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
	"fmt"
	"sync"
	"testing"
	"time"

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
		wait := app.Wait()
		defer app.RequireStart().RequireStop()
		assert.NoError(t, s.Shutdown(), "error returned from first shutdown call")

		assert.EqualError(t, s.Shutdown(), "send terminated signal: 2/2 channels are blocked",
			"unexpected error returned when shutdown is called with a blocked channel")
		assert.NotNil(t, <-done, "done channel did not receive signal")
		assert.NotNil(t, <-wait, "wait channel did not receive signal")
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

	t.Run("with exit code", func(t *testing.T) {
		t.Parallel()
		var s fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&s),
		)

		require.NoError(t, app.Start(context.Background()), "error starting app")
		assert.NoError(t, s.Shutdown(fx.ExitCode(2)), "error in app shutdown")
		wait := <-app.Wait()
		defer app.Stop(context.Background())
		require.Equal(t, 2, wait.ExitCode)
	})

	t.Run("with exit code and multiple Wait", func(t *testing.T) {
		t.Parallel()
		var s fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&s),
		)

		require.NoError(t, app.Start(context.Background()), "error starting app")
		defer require.NoError(t, app.Stop(context.Background()))

		for i := 0; i < 10; i++ {
			t.Run(fmt.Sprintf("Wait %v", i), func(t *testing.T) {
				t.Parallel()
				wait := <-app.Wait()
				require.Equal(t, 2, wait.ExitCode)
			})
		}

		assert.NoError(t, s.Shutdown(fx.ExitCode(2), fx.ShutdownTimeout(time.Second)))
	})

	t.Run("from invoke", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(
			t,
			fx.Invoke(func(s fx.Shutdowner) {
				s.Shutdown()
			}),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(t, app.Start(ctx), "error starting app")
		defer app.Stop(ctx)
		select {
		case <-ctx.Done():
			assert.Fail(t, "app did not shutdown in time")
		case <-app.Wait():
			// success
		}
	})

	t.Run("many times", func(t *testing.T) {
		t.Parallel()

		var shutdowner fx.Shutdowner
		app := fxtest.New(
			t,
			fx.Populate(&shutdowner),
		)

		for i := 0; i < 10; i++ {
			app.RequireStart()
			shutdowner.Shutdown(fx.ExitCode(i))
			assert.Equal(t, i, (<-app.Wait()).ExitCode, "run %d", i)
			app.RequireStop()
		}
	})
}

func TestDataRace(t *testing.T) {
	t.Parallel()

	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Populate(&s),
	)
	defer app.RequireStart().RequireStop()

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
