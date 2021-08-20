// Copyright (c) 2020 Uber Technologies, Inc.
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
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/fx/internal/fxlog"
)

func TestSupply(t *testing.T) {
	type A struct{}
	type B struct{}

	t.Run("NothingIsSupplied", func(t *testing.T) {
		app := fxtest.New(t, fx.Supply())
		defer app.RequireStart().RequireStop()
	})

	t.Run("SomethingIsSupplied", func(t *testing.T) {
		aIn, bIn := &A{}, &B{}
		var aOut *A
		var bOut *B

		app := fxtest.New(
			t,
			fx.Supply(aIn, bIn),
			fx.Populate(&aOut, &bOut),
		)
		defer app.RequireStart().RequireStop()

		require.Same(t, aIn, aOut)
		require.Same(t, bIn, bOut)
	})

	t.Run("AnnotateIsSupplied", func(t *testing.T) {
		firstIn, secondIn, thirdIn := &A{}, &A{}, &B{}
		var out struct {
			fx.In

			First  *A `name:"first"`
			Second *A `name:"second"`
			Third  *B
		}

		app := fxtest.New(
			t,
			fx.Supply(
				fx.Annotated{Name: "first", Target: firstIn},
				fx.Annotated{Name: "second", Target: secondIn},
				thirdIn),
			fx.Populate(&out),
		)
		defer app.RequireStart().RequireStop()

		require.Same(t, firstIn, out.First)
		require.Same(t, secondIn, out.Second)
		require.Same(t, thirdIn, out.Third)
	})

	t.Run("InvalidArgumentIsSupplied", func(t *testing.T) {
		require.PanicsWithValuef(
			t,
			"untyped nil passed to fx.Supply",
			func() { fx.Supply(A{}, nil) },
			"a naked nil should panic",
		)

		require.NotPanicsf(
			t,
			func() { fx.Supply(A{}, (*B)(nil)) },
			"a wrapped nil should not panic",
		)

		require.PanicsWithValuef(
			t,
			"error value passed to fx.Supply",
			func() { fx.Supply(A{}, errors.New("fail")) },
			"an error value should panic",
		)
	})

	t.Run("SupplyCollision", func(t *testing.T) {
		type foo struct{}

		var spy fxlog.Spy
		app := fx.New(
			fx.WithLogger(func() fxevent.Logger { return &spy }),
			fx.Supply(&foo{}, &foo{}),
		)

		require.Error(t, app.Err())
		assert.Contains(t, app.Err().Error(), "already provided")

		supplied := spy.Events().SelectByTypeName("Supplied")
		require.Len(t, supplied, 2)
		require.NoError(t, supplied[0].(*fxevent.Supplied).Err)
		require.Error(t, supplied[1].(*fxevent.Supplied).Err)
	})
}
