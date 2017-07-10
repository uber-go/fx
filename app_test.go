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

package fx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	t.Run("InitializesFields", func(t *testing.T) {
		app := New()
		assert.NotNil(t, app.container)
		assert.NotNil(t, app.lifecycle)
		assert.NotNil(t, app.logger)
	})

	t.Run("ProvidesLifecycle", func(t *testing.T) {
		found := false
		app := New(Invoke(func(lc Lifecycle) {
			assert.NotNil(t, lc)
			found = true
		}))
		require.NoError(t, app.Start(context.Background()))
		assert.True(t, found)
	})

	t.Run("OptionsHappensBeforeProvides", func(t *testing.T) {
		optionRun := false

		p := func() struct{} {
			assert.True(t, optionRun, "Option must run before provides")
			return struct{}{}
		}
		inv := func(struct{}) {}

		anOption := optionFunc(func(app *App) {
			optionRun = true
		})

		app := New(Provide(p), Invoke(inv), anOption)
		app.Start(Timeout(1 * time.Second))
	})
}

func TestOptions(t *testing.T) {
	t.Run("OptionsComposition", func(t *testing.T) {
		var n int
		construct := func() struct{} {
			n++
			return struct{}{}
		}
		use := func(struct{}) {
			n++
		}
		opts := Options(Provide(construct), Invoke(use))
		require.NoError(t, New(opts).Start(context.Background()))
		assert.Equal(t, 2, n)
	})

	t.Run("ProvidesCalledInGraphOrder", func(t *testing.T) {
		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		initOrder := 0
		new1 := func() type1 {
			initOrder++
			assert.Equal(t, 1, initOrder)
			return type1{}
		}
		new2 := func(type1) type2 {
			initOrder++
			assert.Equal(t, 2, initOrder)
			return type2{}
		}
		new3 := func(type1, type2) type3 {
			initOrder++
			assert.Equal(t, 3, initOrder)
			return type3{}
		}
		biz := func(s1 type1, s2 type2, s3 type3) {
			initOrder++
			assert.Equal(t, 4, initOrder)
		}
		app := New(
			Provide(new1, new2, new3),
			Invoke(biz),
		)
		app.Start(context.Background())
		assert.Equal(t, 4, initOrder)
	})

	t.Run("ProvidesCalledLazily", func(t *testing.T) {
		count := 0
		newBuffer := func() *bytes.Buffer {
			t.Error("this module should not init: no provided type relies on it")
			return nil
		}
		newEmpty := func() struct{} {
			count++
			return struct{}{}
		}
		app := New(
			Provide(newBuffer, newEmpty),
			Invoke(func(struct{}) { count++ }),
		)
		app.Start(context.Background())
		assert.Equal(t, 2, count)
	})

	t.Run("Error", func(t *testing.T) {
		app := New(
			Provide(&bytes.Buffer{}), // error, not a constructor
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide constructor function")
	})
}

func TestAppStart(t *testing.T) {
	t.Run("Timeout", func(t *testing.T) {
		block := func() { select {} }
		app := New(Invoke(block))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := app.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StartError", func(t *testing.T) {
		failStart := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStart: func() error {
				return errors.New("OnStart fail")
			}})
			return struct{}{}
		}
		app := New(
			Provide(failStart),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStart fail")
	})

	t.Run("StartAndStopErrors", func(t *testing.T) {
		fail := func(lc Lifecycle) struct{} {
			lc.Append(Hook{
				OnStart: func() error { return errors.New("OnStart fail") },
				OnStop:  func() error { return errors.New("OnStop fail") },
			})
			return struct{}{}
		}
		app := New(
			Provide(fail),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStart fail")
		assert.Contains(t, err.Error(), "OnStop fail")
	})
}

func TestAppStop(t *testing.T) {
	t.Run("Timeout", func(t *testing.T) {
		block := func() error { select {} }
		app := New(Invoke(func(l Lifecycle) {
			l.Append(Hook{OnStop: block})
		}))
		require.NoError(t, app.Start(context.Background()))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := app.Stop(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StopError", func(t *testing.T) {
		failStop := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStop: func() error {
				return errors.New("OnStop fail")
			}})
			return struct{}{}
		}
		app := New(
			Provide(failStop),
			Invoke(func(struct{}) {}),
		)
		assert.NoError(t, app.Start(context.Background()))
		err := app.Stop(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStop fail")
	})
}

func TestAppRun(t *testing.T) {
	run := func(app *App) {
		go func() {
			app.Run()
		}()
		// 100ms sleep makes sure that app is in RUNNING state
		time.Sleep(100 * time.Millisecond)
	}
	t.Run("BlocksOnSignals", func(t *testing.T) {
		app := New()

		proc, err := os.FindProcess(os.Getpid())
		assert.NoError(t, err, "Could not find process")

		run(app)
		assert.NoError(t, proc.Signal(os.Interrupt), "Could not send os.Interrupt")

		run(app)
		assert.NoError(t, proc.Signal(syscall.SIGTERM), "Could not send syscall.SIGTERM")
	})
}

type printerSpy struct {
	*bytes.Buffer
}

func (ps printerSpy) Printf(format string, args ...interface{}) {
	fmt.Fprintf(ps.Buffer, format, args...)
	ps.Buffer.WriteRune('\n')
}

func TestReplaceLogger(t *testing.T) {
	spy := printerSpy{&bytes.Buffer{}}
	app := New(Logger(spy))
	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, app.Stop(context.Background()))
	assert.Contains(t, spy.String(), "RUNNING")
}
