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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestSupply(t *testing.T) {
	type A struct{}
	type B struct{}

	t.Run("NothingIsSupplied", func(t *testing.T) {
		app := fxtest.New(t, fx.Supply())
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("SomethingIsSupplied", func(t *testing.T) {
		aIn, bIn := A{}, B{}
		aOut, bOut := (*A)(nil), (*B)(nil)
		app := fxtest.New(t, fx.Supply(&aIn, &bIn), fx.Populate(&aOut, &bOut))
		defer app.RequireStart().RequireStop()

		require.NoError(t, app.Err())
		require.Equal(t, &aIn, aOut)
		require.Equal(t, &bIn, bOut)
	})

	t.Run("InvalidArgumentIsSupplied", func(t *testing.T) {
		require.PanicsWithValuef(
			t, "untyped nil passed to fx.Supply",
			func() { fx.Supply(A{}, nil) },
			"a naked nil should panic")

		require.NotPanicsf(
			t, func() {
				fx.Supply(A{}, (*B)(nil))
			},
			"a non-naked nil should not panic")

		require.PanicsWithValuef(
			t, "error value passed to fx.Supply",
			func() { fx.Supply(A{}, errors.New("fail")) },
			"an error value should panic")
	})
}
