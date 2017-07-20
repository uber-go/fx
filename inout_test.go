package fx_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestIn(t *testing.T) {
	t.Run("IsDigIn", func(t *testing.T) {
		t.Parallel()

		type in struct {
			fx.In
		}
		assert.True(t, dig.IsIn(reflect.TypeOf(in{})), "Expected dig.In to work with fx.In")
	})
	t.Run("Optionals", func(t *testing.T) {
		t.Parallel()

		type foo struct{}
		newFoo := func() *foo { return &foo{} }

		type bar struct{}
		newBar := func() *bar { return &bar{} }

		type params struct {
			fx.In

			Foo *foo
			Bar *bar `optional:"true"`
		}

		t.Run("NotProvided", func(t *testing.T) {
			t.Parallel()

			ran := false
			app := fxtest.New(t, fx.Provide(newFoo), fx.Invoke(func(params params) {
				assert.NotNil(t, params.Foo, "foo was not optional and provided, expected not nil")
				assert.Nil(t, params.Bar, "bar was optional and not provided, expected nil")
				ran = true
			}))
			app.MustStart().MustStop()
			assert.True(t, ran, "expected invoke to run")
		})

		t.Run("Provided", func(t *testing.T) {
			t.Parallel()

			ran := false
			app := fxtest.New(t, fx.Provide(newFoo, newBar), fx.Invoke(func(params params) {
				assert.NotNil(t, params.Foo, "foo was not optional and provided, expected not nil")
				assert.NotNil(t, params.Bar, "bar was optional and provided, expected not nil")
				ran = true
			}))
			app.MustStart().MustStop()
			assert.True(t, ran, "expected invoke to run")
		})
	})
}

func TestOut(t *testing.T) {
	t.Run("IsDigOut", func(t *testing.T) {
		t.Parallel()

		type out struct {
			fx.Out
		}
		assert.True(t, dig.IsOut(reflect.TypeOf(out{})), "expected dig.Out to work with fx.Out")
	})
}

func TestNamed(t *testing.T) {
	t.Parallel()

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

	// another constructor that returns the type a with name "bar"
	type barOut struct {
		fx.Out
		A *a `name:"bar"`
	}
	newBar := func() barOut {
		return barOut{
			A: &a{name: "bar"},
		}
	}

	t.Run("InjectFoo", func(t *testing.T) {
		t.Parallel()

		// an invoke expects to get type a with name "foo"
		type fooIn struct {
			fx.In

			A *a `name:"foo"`
		}
		ran := false
		app := fxtest.New(t, fx.Provide(newFoo, newBar), fx.Invoke(func(in fooIn) {
			assert.NotNil(t, in.A, "expected in.a to be injected")
			assert.Equal(t, "foo", in.A.name, "expected to get type a of name foo")
			ran = true
		}))
		app.MustStart().MustStop()
		assert.True(t, ran, "expected invoke to run")

	})

	t.Run("InjectBar", func(t *testing.T) {
		t.Parallel()

		// another expects to get type a with name "bar"
		type barIn struct {
			fx.In

			A *a `name:"bar"`
		}
		ran := false
		app := fxtest.New(t, fx.Provide(newFoo, newBar), fx.Invoke(func(in barIn) {
			assert.NotNil(t, in.A, "expected in.a to be injected")
			assert.Equal(t, "bar", in.A.name, "expected to get type a of name bar")
			ran = true
		}))
		app.MustStart().MustStop()
		assert.True(t, ran, "expected invoke to run")
	})
}