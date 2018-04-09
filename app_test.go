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

package fx_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/multierr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type printerSpy struct {
	*bytes.Buffer
}

func (ps printerSpy) Printf(format string, args ...interface{}) {
	fmt.Fprintf(ps.Buffer, format, args...)
	ps.Buffer.WriteRune('\n')
}

func TestNewApp(t *testing.T) {
	t.Run("ProvidesLifecycle", func(t *testing.T) {
		found := false
		app := fxtest.New(t, Invoke(func(lc Lifecycle) {
			assert.NotNil(t, lc)
			found = true
		}))
		defer app.RequireStart().RequireStop()
		assert.True(t, found)
	})

	t.Run("OptionsHappensBeforeProvides", func(t *testing.T) {
		// Should be grouping all provides and pushing them into the container
		// after applying other options. This prevents the app configuration
		// (e.g., logging) from changing halfway through our provides.

		spy := &printerSpy{&bytes.Buffer{}}
		app := fxtest.New(t, Provide(func() struct{} { return struct{}{} }), Logger(spy))
		defer app.RequireStart().RequireStop()
		assert.Contains(t, spy.String(), "PROVIDE\tstruct {}")
	})

	t.Run("CircularGraphReturnsError", func(t *testing.T) {
		type A struct{}
		type B struct{}
		app := fxtest.New(t,
			Provide(func(A) B { return B{} }),
			Provide(func(B) A { return A{} }),
		)
		err := app.Err()
		require.Error(t, err, "fx.New should return an error")
		assert.Contains(t, err.Error(), "fx_test.A -> fx_test.B -> fx_test.A")
	})
}

func TestInvokes(t *testing.T) {
	t.Run("ErrorsAreNotOverriden", func(t *testing.T) {
		type A struct{}
		type B struct{}

		app := fxtest.New(t,
			Provide(func() B { return B{} }), // B inserted into the graph
			Invoke(func(A) {}),               // failed A invoke
			Invoke(func(B) {}),               // successful B invoke
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "A isn't in the container")
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
		app := fxtest.New(t, Options(Provide(construct), Invoke(use)))
		defer app.RequireStart().RequireStop()
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
		app := fxtest.New(t,
			Provide(new1, new2, new3),
			Invoke(biz),
		)
		defer app.RequireStart().RequireStop()
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
		app := fxtest.New(t,
			Provide(newBuffer, newEmpty),
			Invoke(func(struct{}) { count++ }),
		)
		defer app.RequireStart().RequireStop()
		assert.Equal(t, 2, count)
	})

	t.Run("Error", func(t *testing.T) {
		spy := printerSpy{&bytes.Buffer{}}
		fxtest.New(t,
			Provide(&bytes.Buffer{}), // error, not a constructor
			Logger(spy),
		)
		assert.Contains(t, spy.String(), "must provide constructor function")
	})
}

func TestTimeoutOptions(t *testing.T) {
	const timeout = time.Minute
	// Further assertions can't succeed unless the test timeout is greater than the default.
	require.True(t, timeout > DefaultTimeout, "test assertions require timeout greater than default")

	var started, stopped bool
	assertCustomContext := func(ctx context.Context, phase string) {
		deadline, ok := ctx.Deadline()
		if assert.True(t, ok, "no %s deadline", phase) {
			remaining := time.Until(deadline)
			assert.True(t, remaining > DefaultTimeout, "didn't respect customized %s timeout", phase)
		}
	}
	verify := func(lc Lifecycle) {
		lc.Append(Hook{
			OnStart: func(ctx context.Context) error {
				assertCustomContext(ctx, "start")
				started = true
				return nil
			},
			OnStop: func(ctx context.Context) error {
				assertCustomContext(ctx, "stop")
				stopped = true
				return nil
			},
		})
	}
	app := fxtest.New(
		t,
		Invoke(verify),
		StartTimeout(timeout),
		StopTimeout(timeout),
	)

	app.RequireStart().RequireStop()
	assert.True(t, started, "app wasn't started")
	assert.True(t, stopped, "app wasn't stopped")
}

func TestAppStart(t *testing.T) {
	t.Run("Timeout", func(t *testing.T) {
		type A struct{}
		blocker := func(lc Lifecycle) *A {
			lc.Append(
				Hook{
					OnStart: func(context.Context) error {
						select {}
					},
				},
			)
			return &A{}
		}
		app := fxtest.New(
			t,
			Provide(blocker),
			Invoke(func(*A) {}),
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := app.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StartError", func(t *testing.T) {
		failStart := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStart: func(context.Context) error {
				return errors.New("OnStart fail")
			}})
			return struct{}{}
		}
		app := fxtest.New(t,
			Provide(failStart),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStart fail")
	})

	t.Run("StartAndStopErrors", func(t *testing.T) {
		errStop1 := errors.New("OnStop fail 1")
		errStart2 := errors.New("OnStart fail 2")
		fail := func(lc Lifecycle) struct{} {
			lc.Append(Hook{
				OnStart: func(context.Context) error { return nil },
				OnStop:  func(context.Context) error { return errStop1 },
			})
			lc.Append(Hook{
				OnStart: func(context.Context) error { return errStart2 },
				OnStop:  func(context.Context) error { assert.Fail(t, "should be never called"); return nil },
			})
			return struct{}{}
		}
		app := fxtest.New(t,
			Provide(fail),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Equal(t, []error{errStart2, errStop1}, multierr.Errors(err))
	})

	t.Run("InvokeNonFunction", func(t *testing.T) {
		app := fxtest.New(t, Invoke(struct{}{}))
		err := app.Start(context.Background())
		require.Error(t, err, "expected start failure")
		assert.Contains(t, err.Error(), "can't invoke non-function")
	})

	t.Run("ProvidingAProvideShouldFail", func(t *testing.T) {
		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		app := fxtest.New(t,
			Provide(
				func() type1 { return type1{} },
				Provide(
					func() type2 { return type2{} },
					func() type3 { return type3{} },
				),
			),
		)

		err := app.Start(context.Background())
		require.Error(t, err, "expected start failure")
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Provide")
		assert.Contains(t, err.Error(), "fx.Provide received fx.Provide(go.uber.org/fx_test.TestAppStart")
	})

	t.Run("InvokingAnInvokeShouldFail", func(t *testing.T) {
		type type1 struct{}

		app := fxtest.New(t,
			Provide(func() type1 { return type1{} }),
			Invoke(Invoke(func(type1) {
			})),
		)
		newErr := app.Err()
		require.Error(t, newErr)

		err := app.Start(context.Background())
		require.Error(t, err, "expected start failure")
		assert.Equal(t, err, newErr, "start should return the same error fx.New encountered")

		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Invoke")
		assert.Contains(t, err.Error(), "fx.Invoke received fx.Invoke(go.uber.org/fx_test.TestAppStart")
	})

	t.Run("ProvidingOptionsShouldFail", func(t *testing.T) {
		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		module := Options(
			Provide(
				func() type1 { return type1{} },
				func() type2 { return type2{} },
			),
			Invoke(func(type1) {
				require.FailNow(t, "module Invoke must not be called")
			}),
		)

		app := fxtest.New(t,
			Provide(
				func() type3 { return type3{} },
				module,
			),
		)
		err := app.Start(context.Background())
		require.Error(t, err, "expected start failure")
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Provide")
		assert.Contains(t, err.Error(), "fx.Provide received fx.Options(fx.Provide(go.uber.org/fx_test.TestAppStart")
	})
}

func TestAppStop(t *testing.T) {
	t.Run("Timeout", func(t *testing.T) {
		block := func(context.Context) error { select {} }
		app := fxtest.New(t, Invoke(func(l Lifecycle) {
			l.Append(Hook{OnStop: block})
		}))
		app.RequireStart()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := app.Stop(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StopError", func(t *testing.T) {
		failStop := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStop: func(context.Context) error {
				return errors.New("OnStop fail")
			}})
			return struct{}{}
		}
		app := fxtest.New(t,
			Provide(failStop),
			Invoke(func(struct{}) {}),
		)
		app.RequireStart()
		err := app.Stop(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStop fail")
	})
}

func TestDone(t *testing.T) {
	done := fxtest.New(t).Done()
	require.NotNil(t, done, "Got a nil channel.")
	select {
	case sig := <-done:
		t.Fatalf("Got unexpected signal %v from application's Done channel.", sig)
	default:
	}
}

func TestReplaceLogger(t *testing.T) {
	spy := printerSpy{&bytes.Buffer{}}
	app := fxtest.New(t, Logger(spy))
	app.RequireStart().RequireStop()
	assert.Contains(t, spy.String(), "RUNNING")
}

func TestNopLogger(t *testing.T) {
	app := fxtest.New(t, NopLogger)
	app.RequireStart().RequireStop()
}
