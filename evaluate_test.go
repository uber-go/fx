// Copyright (c) 2025 Uber Technologies, Inc.
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
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestEvaluate(t *testing.T) {
	t.Run("ProvidesOptions", func(t *testing.T) {
		type t1 struct{}
		type t2 struct{}

		var evaluated, provided, invoked bool
		app := fxtest.New(t,
			fx.Evaluate(func() fx.Option {
				evaluated = true
				return fx.Provide(func() t1 {
					provided = true
					return t1{}
				})
			}),
			fx.Provide(func(t1) t2 { return t2{} }),
			fx.Invoke(func(t2) {
				invoked = true
			}),
		)
		defer app.RequireStart().RequireStop()

		assert.True(t, evaluated, "Evaluated function was not called")
		assert.True(t, provided, "Provided function was not called")
		assert.True(t, invoked, "Invoked function was not called")
	})

	t.Run("OptionalDependency", func(t *testing.T) {
		type Config struct{ Dev bool }

		newBufWriter := func(b *bytes.Buffer) io.Writer {
			return b
		}

		newDiscardWriter := func() io.Writer {
			return io.Discard
		}

		newWriter := func(cfg Config) fx.Option {
			if cfg.Dev {
				return fx.Provide(newDiscardWriter)
			}

			return fx.Provide(newBufWriter)
		}

		t.Run("NoDependency", func(t *testing.T) {
			var got io.Writer
			app := fxtest.New(t,
				fx.Evaluate(newWriter),
				fx.Provide(
					func() *bytes.Buffer {
						t.Errorf("unexpected call to *bytes.Buffer")
						return nil
					},
				),
				fx.Supply(Config{Dev: true}),
				fx.Populate(&got),
			)
			defer app.RequireStart().RequireStop()

			assert.NotNil(t, got)
			_, _ = io.WriteString(got, "hello")
		})

		t.Run("WithDependency", func(t *testing.T) {
			var (
				buf bytes.Buffer
				got io.Writer
			)
			app := fxtest.New(t,
				fx.Evaluate(newWriter),
				fx.Supply(&buf, Config{Dev: false}),
				fx.Populate(&got),
			)
			defer app.RequireStart().RequireStop()

			assert.NotNil(t, got)
			_, _ = io.WriteString(got, "hello")
			assert.Equal(t, "hello", buf.String())
		})
	})
}
