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

// leaky_test contains tests that are explicitly leaking goroutines because they
// trigger a panic on purpose.
// To prevent these from making it difficult for us to use goleak for tests in
// fx_test, we contain these here.
package leaky_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestRecoverFromPanicsOption(t *testing.T) {
	t.Parallel()

	t.Run("CannotGiveToModule", func(t *testing.T) {
		t.Parallel()
		mod := fx.Module("MyModule", fx.RecoverFromPanics())
		err := fx.New(mod).Err()
		require.Error(t, err)
		require.Contains(t, err.Error(),
			"fx.RecoverFromPanics Option should be passed to top-level App, "+
				"not to fx.Module")
	})
	run := func(withOption bool) {
		opts := []fx.Option{
			fx.Provide(func() int {
				panic("terrible sorrow")
			}),
			fx.Invoke(func(a int) {}),
		}
		if withOption {
			opts = append(opts, fx.RecoverFromPanics())
			app := fx.New(opts...)
			err := app.Err()
			require.Error(t, err)
			assert.Contains(t, err.Error(),
				`panic: "terrible sorrow" in func: "go.uber.org/fx/internal/leaky_test_test".TestRecoverFromPanicsOption.`)
		} else {
			assert.Panics(t, func() { fx.New(opts...) },
				"expected panic without RecoverFromPanics() option")
		}
	}

	t.Run("WithoutOption", func(t *testing.T) { run(false) })
	t.Run("WithOption", func(t *testing.T) { run(true) })
}
