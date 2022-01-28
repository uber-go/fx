// Copyright (c) 2022 Uber Technologies, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestModuleSuccess(t *testing.T) {
	t.Parallel()

	t.Run("provide a dependency from a submodule", func(t *testing.T) {
		t.Parallel()
		type Logger struct {
			Name string
		}

		redis := fx.Module("redis",
			fx.Provide(func() *Logger {
				return &Logger{Name: "redis"}
			}),
		)

		app := fxtest.New(t,
			redis,
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, l.Name, "redis")
			}),
		)

		defer app.RequireStart().RequireStop()
	})

	t.Run("provide a dependency from nested modules", func(t *testing.T) {
		t.Parallel()
		type Logger struct {
			Name string
		}

		grandchild := fx.Module("grandchild",
			fx.Provide(func() *Logger {
				return &Logger{Name: "redis"}
			}),
		)

		child := fx.Module("child",
			grandchild,
		)

		app := fxtest.New(t,
			child,
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, l.Name, "redis")
			}),
		)
		defer app.RequireStart().RequireStop()
	})
}

func TestModuleFailures(t *testing.T) {
	t.Parallel()

	t.Run("invoke from submodule failed", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		sub := fx.Module("sub",
			fx.Provide(func() *A { return &A{} }),
			fx.Invoke(func(*A, *B) { // missing dependency.
				require.Fail(t, "this should not be called")
			}),
		)

		app := NewForTest(t,
			sub,
			fx.Invoke(func(a *A) {
				assert.NotNil(t, a)
			}),
		)

		err := app.Err()
		require.Error(t, err)
	})

	t.Run("provide the same dependency from multiple modules", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		app := NewForTest(t,
			fx.Provide(func() A { return A{} }),
			fx.Provide(func() A { return A{} }),
			fx.Invoke(func(a A) {}),
		)

		err := app.Err()
		require.Error(t, err)
	})
}
