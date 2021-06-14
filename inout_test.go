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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestIn(t *testing.T) {
	type in struct {
		fx.In
	}
	assert.True(t, dig.IsIn(in{}), "Expected dig.In to work with fx.In")
}

func TestOut(t *testing.T) {
	type out struct {
		fx.Out
	}
	assert.True(t, dig.IsOut(out{}), "expected dig.Out to work with fx.Out")
}

func TestOptionalTypes(t *testing.T) {
	type foo struct{}
	newFoo := func() *foo { return &foo{} }

	type bar struct{}
	newBar := func() *bar { return &bar{} }

	type in struct {
		fx.In

		Foo *foo
		Bar *bar `optional:"true"`
	}

	t.Run("NotProvided", func(t *testing.T) {
		ran := false
		app := fxtest.New(t, fx.Provide(newFoo), fx.Invoke(func(in in) {
			assert.NotNil(t, in.Foo, "foo was not optional and provided, expected not nil")
			assert.Nil(t, in.Bar, "bar was optional and not provided, expected nil")
			ran = true
		}))
		app.RequireStart().RequireStop()
		assert.True(t, ran, "expected invoke to run")
	})

	t.Run("Provided", func(t *testing.T) {
		ran := false
		app := fxtest.New(t, fx.Provide(newFoo, newBar), fx.Invoke(func(in in) {
			assert.NotNil(t, in.Foo, "foo was not optional and provided, expected not nil")
			assert.NotNil(t, in.Bar, "bar was optional and provided, expected not nil")
			ran = true
		}))
		app.RequireStart().RequireStop()
		assert.True(t, ran, "expected invoke to run")
	})
}

func TestNamedTypes(t *testing.T) {
	type a struct {
		name string
	}

	// a constructor that returns the type a with name "foo"
	type fooOut struct {
		fx.Out

		A *a `name:"foo"`
	}
	newFoo := func() fooOut {
		return fooOut{
			A: &a{name: "foo"},
		}
	}

	// another constructor that returns the same type a with name "bar"
	type barOut struct {
		fx.Out

		A *a `name:"bar"`
	}
	newBar := func() barOut {
		return barOut{
			A: &a{name: "bar"},
		}
	}

	// invoke with an fx.In that resolves both named types
	type in struct {
		fx.In

		Foo *a `name:"foo"`
		Bar *a `name:"bar"`
	}
	ran := false
	app := fxtest.New(t, fx.Provide(newFoo, newBar), fx.Invoke(func(in in) {
		assert.NotNil(t, in.Foo, "expected in.Foo to be injected")
		assert.Equal(t, "foo", in.Foo.name, "expected to get type 'a' of name 'foo'")

		assert.NotNil(t, in.Bar, "expected in.Bar to be injected")
		assert.Equal(t, "bar", in.Bar.name, "expected to get a type 'a' of name 'bar'")

		ran = true
	}))
	app.RequireStart().RequireStop()
	assert.True(t, ran, "expected invoke to run")
}

func TestIgnoreUnexported(t *testing.T) {
	type A struct{ ID int }
	type B struct{ ID int }

	type Params struct {
		fx.In `ignore-unexported:"true"`

		A A
		b B // will be ignored
	}

	ran := false
	run := func(in Params) {
		defer func() { ran = true }()

		assert.Equal(t, A{1}, in.A, "A must be set")

		// We provide a B to the container, but because the "b" field
		// is unexported, we don't expect it to be set.
		assert.Equal(t, B{0}, in.b, "b must be unset")
	}
	defer func() {
		assert.True(t, ran, "run was never called")
	}()

	fxtest.New(t,
		fx.Supply(A{1}, B{2}),
		fx.Invoke(run),
	).RequireStart().RequireStop()
}
