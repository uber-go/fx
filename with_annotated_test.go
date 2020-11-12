package fx_test

import (
	"testing"

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
			fx.Invoke(fx.WithAnnotated("foo")(func(aa *a, bb *b) {
				inA = aa
				inB = bb
			})),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, inA, "expected a to be injected")
		assert.NotNil(t, inB, "expected b to be injected")
		assert.Equal(t, "foo", inA.name, "expected to get a type 'a' of name 'foo'")
		assert.Equal(t, "bar", inB.name, "expected to get a type 'b' of name 'bar'")
	})
}
