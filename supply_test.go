package fx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestSupply(t *testing.T) {
	t.Run("NothingIsSupplied", func(t *testing.T) {
		app := fxtest.New(t, fx.Supply())
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("SomethingIsSupplied", func(t *testing.T) {
		type A struct{}
		type B struct{}

		aIn, bIn := A{}, B{}
		aOut, bOut := (*A)(nil), (*B)(nil)
		app := fxtest.New(t, fx.Supply(&aIn, &bIn), fx.Populate(&aOut, &bOut))
		defer app.RequireStart().RequireStop()

		require.NoError(t, app.Err())
		require.Equal(t, &aIn, aOut)
		require.Equal(t, &bIn, bOut)
	})
}
