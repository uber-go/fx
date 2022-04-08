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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestReplaceSuccess(t *testing.T) {
	t.Parallel()

	t.Run("replace a value", func(t *testing.T) {
		t.Parallel()
		type A struct {
			Value string
		}
		a := &A{Value: "a'"}
		app := fxtest.New(t,
			fx.Provide(func() *A {
				return &A{
					Value: "a",
				}
			}),
			fx.Replace(a),
			fx.Invoke(func(a *A) {
				assert.Equal(t, "a'", a.Value)
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("replace in a module", func(t *testing.T) {
		t.Parallel()

		type A struct {
			Value string
		}

		a := &A{Value: "A"}

		app := fxtest.New(t,
			fx.Module("child",
				fx.Replace(a),
				fx.Invoke(func(a *A) {
					assert.Equal(t, "A", a.Value)
				}),
			),
			fx.Provide(func() *A {
				return &A{
					Value: "a",
				}
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("replace with annotate", func(t *testing.T) {
		t.Parallel()

		type A struct {
			Value string
		}

		app := fxtest.New(t,
			fx.Supply(
				fx.Annotate(A{"A"}, fx.ResultTags(`name:"t"`)),
			),
			fx.Replace(
				fx.Annotate(A{"B"}, fx.ResultTags(`name:"t"`)),
			),
			fx.Invoke(fx.Annotate(func(a A) {
				assert.Equal(t, a.Value, "B")
			}, fx.ParamTags(`name:"t"`))),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("replace a value group with annotate", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Supply(
				fx.Annotate([]string{"A", "B", "C"}, fx.ResultTags(`group:"t,flatten"`)),
			),
			fx.Replace(fx.Annotate([]string{"a", "b", "c"}, fx.ResultTags(`group:"t"`))),
			fx.Invoke(fx.Annotate(func(ss ...string) {
				assert.ElementsMatch(t, []string{"a", "b", "c"}, ss)
			}, fx.ParamTags(`group:"t"`))),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("replace a value group supplied by a child module from root module", func(t *testing.T) {
		t.Parallel()

		foo := fx.Module("foo",
			fx.Supply(
				fx.Annotate([]string{"a", "b", "c"}, fx.ResultTags(`group:"t,flatten"`)),
			),
		)

		fxtest.New(t,
			fx.Module("wrapfoo",
				foo,
				fx.Replace(
					fx.Annotate([]string{"d", "e", "f"}, fx.ResultTags(`group:"t"`)),
				),
				fx.Invoke(fx.Annotate(func(ss []string) {
					assert.ElementsMatch(t, []string{"d", "e", "f"}, ss)
				}, fx.ParamTags(`group:"t"`))),
			),
		)
	})
}

func TestReplaceFailure(t *testing.T) {
	t.Parallel()

	t.Run("replace same value twice", func(t *testing.T) {
		t.Parallel()

		type A struct {
			Value string
		}
		a := &A{Value: "A"}
		app := NewForTest(t,
			fx.Provide(func() *A {
				return &A{Value: "a"}
			}),
			fx.Module("child",
				fx.Replace(a),
				fx.Replace(a),
				fx.Invoke(func(a *A) {
					assert.Fail(t, "this should never run")
				}),
			),
		)
		err := app.Err()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "*fx_test.A already decorated")
	})

	t.Run("replace a value that wasn't provided", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		app := NewForTest(t,
			fx.Replace(A{}),
			fx.Invoke(func(a *A) {
			}),
		)
		err := app.Err()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing type: *fx_test.A")
	})

	t.Run("replace panics on invalid values", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		require.PanicsWithValuef(
			t,
			"untyped nil passed to fx.Replace",
			func() { fx.Replace(A{}, nil) },
			"a naked nil should panic",
		)

		require.PanicsWithValuef(
			t,
			"error value passed to fx.Replace",
			func() { fx.Replace(A{}, errors.New("some error")) },
			"replacing with an error should panic",
		)

		require.NotPanicsf(
			t,
			func() { fx.Replace(A{}, (*B)(nil)) },
			"a wrapped nil should not panic")
	})
}
