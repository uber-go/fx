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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type type1 struct{}
type type2 struct{}
type type3 struct{}

func TestApp(t *testing.T) {
	t.Run("NewCreatesApp", func(t *testing.T) {
		s := New()
		assert.NotNil(t, s.container)
		assert.NotNil(t, s.lifecycle)
	})
	t.Run("NewProvidesLifecycle", func(t *testing.T) {
		found := false
		s := New()
		err := s.Start(context.Background(),
			func(lifecycle Lifecycle) error {
				assert.NotNil(t, lifecycle)
				found = true
				return nil
			})

		require.NoError(t, err)
		assert.True(t, found)
	})
	t.Run("InitsInOrder", func(t *testing.T) {
		initOrder := 0
		new1 := func() *type1 {
			initOrder++
			assert.Equal(t, 1, initOrder)
			return &type1{}
		}
		new2 := func(*type1) *type2 {
			initOrder++
			assert.Equal(t, 2, initOrder)
			return &type2{}
		}
		new3 := func(*type1, *type2) *type3 {
			initOrder++
			assert.Equal(t, 3, initOrder)
			return &type3{}
		}
		biz := func(s1 *type1, s2 *type2, s3 *type3) error {
			initOrder++
			assert.Equal(t, 4, initOrder)
			return nil
		}
		s := New()
		s.Provide(new1, new2, new3)
		s.Start(context.Background(), biz)
		assert.Equal(t, 4, initOrder)
	})
	t.Run("ModulesLazyInit", func(t *testing.T) {
		count := 0
		new1 := func() *type1 {
			t.Error("this module should not init: no provided type relies on it")
			return nil
		}
		new2 := func() *type2 {
			count++
			return &type2{}
		}
		new3 := func(*type2) *type3 {
			count++
			return &type3{}
		}
		biz := func(s2 *type2, s3 *type3) error {
			count++
			return nil
		}
		s := New()
		s.Provide(new1, new2, new3)        // these are lazy loaded
		s.Start(context.Background(), biz) // this is invoked explicitly
		assert.Equal(t, 3, count)
	})
	t.Run("StartRequiresConstructors", func(t *testing.T) {
		s := New()
		err := s.Start(context.Background(), &type1{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "*fx.type1 &{} is not a function")
	})
	t.Run("StartTimeout", func(t *testing.T) {
		s := New()
		startCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		type empty struct{}
		err := s.Start(startCtx, func() (*empty, error) {
			select {}
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
	t.Run("StopTimeout", func(t *testing.T) {
		s := New()
		type empty struct{}
		err := s.Start(context.Background(), func(l Lifecycle) (*empty, error) {
			l.Append(Hook{
				OnStop: func() error {
					select {}
				}})

			return &empty{}, nil
		})
		require.NoError(t, err)

		stopCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		err = s.Stop(stopCtx)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
	t.Run("ProvideDoesNotPanicForObjectInstances", func(t *testing.T) {
		type empty struct{}
		assert.NotPanics(t, func() { New().Provide(&empty{}) })
	})
}

func TestInvoke(t *testing.T) {
	t.Run("CalledAfterStart", func(t *testing.T) {
		s := New()

		var (
			started bool
			gave    *type1
		)

		construct := func(lc Lifecycle) *type1 {
			gave = &type1{}
			lc.Append(Hook{
				OnStart: func() error {
					started = true
					return nil
				},
			})
			return gave
		}

		s.Provide(construct)
		require.NoError(t, s.Start(
			context.Background(),
			func(x *type1) {
				require.NotNil(t, x, "value must not be nil")
				require.True(t, x == gave, "value did not match")
				require.False(t, started, "started too early")
			},
		), "failed to start")
		defer func() {
			assert.NoError(t, s.Stop(context.Background()), "failed to stop")
		}()

		require.NoError(t, s.Invoke(
			func(x *type1) {
				require.NotNil(t, x, "value must not be nil")
				require.True(t, x == gave, "value did not match")
				require.True(t, started, "must be started on invoke")
			},
		), "invoke failed")
	})

	t.Run("Error", func(t *testing.T) {
		s := New()

		calls := 0
		new1 := func() *type1 { calls++; return &type1{} }
		new2 := func() *type2 { calls++; return &type2{} }
		new3 := func() *type3 { calls++; return &type3{} }

		s.Provide(new1, new2, new3)
		require.NoError(t, s.Start(
			context.Background(),
			func(x *type1, y *type2, z *type3) {
				assert.Equal(t, 3, calls, "wrong number of calls")
				require.NotNil(t, x, "type1 was nil")
				require.NotNil(t, y, "type2 was nil")
				require.NotNil(t, z, "type2 was nil")
			},
		), "start failed")
		defer func() {
			assert.NoError(t, s.Stop(context.Background()), "failed to stop")
		}()

		t.Run("FirstFail", func(t *testing.T) {
			err := s.Invoke(
				func(*type1) error {
					return errors.New("great sadness")
				},
				func(*type2) error {
					t.Fatal("second function must not be called")
					return nil
				},
			)
			assert.Equal(t, errors.New("great sadness"), err, "error must match")
		})

		t.Run("SecondFail", func(t *testing.T) {
			err := s.Invoke(
				func(*type1) error {
					return nil
				},
				func(*type2) error {
					return errors.New("great sadness")
				},
				func(*type3) error {
					t.Fatal("third function must not be called")
					return nil
				},
			)
			assert.Equal(t, errors.New("great sadness"), err, "error must match")
		})
	})
}
