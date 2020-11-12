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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestWithAnnotated(t *testing.T) {
	type a struct {
		name string
	}

	type b struct {
		name string
	}

	newA := func() *a {
		return &a{name: "foo"}
	}

	newB := func() *b {
		return &b{name: "bar"}
	}

	t.Run("Provided", func(t *testing.T) {
		var inA *a
		var inB *b
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
				newB,
			),
			fx.Invoke(fx.WithAnnotated("foo")(func(aa *a, bb *b) error {
				inA = aa
				inB = bb
				return nil
			})),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, inA, "expected a to be injected")
		assert.NotNil(t, inB, "expected b to be injected")
		assert.Equal(t, "foo", inA.name, "expected to get a type 'a' of name 'foo'")
		assert.Equal(t, "bar", inB.name, "expected to get a type 'b' of name 'bar'")
	})
}

func TestWithAnnotatedError(t *testing.T) {
	type a struct {
		name string
	}

	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("Provided", func(t *testing.T) {
		app := NewForTest(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
			),
			fx.Invoke(fx.WithAnnotated("foo")("")),
		)
		err := app.Start(context.TODO())
		require.Error(t, err)
		assert.Contains(t, err.Error(), " WithAnnotated returned function must be called with a function")
	})
}
