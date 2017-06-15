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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	type type1 struct{}
	type type2 struct{}
	type type3 struct{}

	t.Run("NewCreatesApp", func(t *testing.T) {
		s := New()
		assert.NotNil(t, s.container)
		assert.NotNil(t, s.lifecycle)
		assert.NotNil(t, s.logger)
	})

	t.Run("NewProvidesLifecycle", func(t *testing.T) {
		found := false
		s := New(Invoke(func(lc Lifecycle) {
			assert.NotNil(t, lc)
			found = true
		}))
		require.NoError(t, s.Start(context.Background()))
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
		s := New(
			Provide(new1, new2, new3),
			Invoke(biz),
		)
		s.Start(context.Background())
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
		s := New(
			Provide(new1, new2, new3),
			Invoke(biz),
		)
		s.Start(context.Background())
		assert.Equal(t, 3, count)
	})

	t.Run("StartTimeout", func(t *testing.T) {
		block := func() { select {} }
		s := New(Invoke(block))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := s.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StopTimeout", func(t *testing.T) {
		block := func() error { select {} }
		s := New(Invoke(func(l Lifecycle) {
			l.Append(Hook{OnStop: block})
		}))
		require.NoError(t, s.Start(context.Background()))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err := s.Stop(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}
