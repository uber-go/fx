package fx_test

import (
	"testing"
	"go.uber.org/fx/fxtest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

func TestAnnotated(t *testing.T) {
	type a struct {
		name string
	}

	type in struct {
		fx.In

		A *a `name:"foo"`
	}
	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("Provided", func(t *testing.T) {
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name: "foo",
					Fn:   newA,
				},
			),
			fx.Invoke(func(in in) {
				assert.NotNil(t, in.A, "expected in.A to be injected")
				assert.Equal(t, "foo", in.A.name, "expected to get a type 'a' of name 'foo'")
			}),
		)
		app.RequireStart().RequireStop()
	})
}
