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

package fx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestDecorateSuccess(t *testing.T) {
	type Logger struct {
		Name string
	}

	t.Run("decorate something from Module", func(t *testing.T) {
		redis := fx.Module("redis",
			fx.Provide(func() *Logger {
				return &Logger{Name: "redis"}
			}),
		)

		testRedis := fx.Module("testRedis",
			redis,
			fx.Decorate(func() *Logger {
				return &Logger{Name: "testRedis"}
			}),
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "testRedis", l.Name)
			}),
		)

		app := fxtest.New(t,
			testRedis,
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "redis", l.Name)
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("decorate a dependency from root", func(t *testing.T) {
		redis := fx.Module("redis",
			fx.Decorate(func() *Logger {
				return &Logger{Name: "redis"}
			}),
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "redis", l.Name)
			}),
		)
		app := fxtest.New(t,
			redis,
			fx.Provide(func() *Logger {
				assert.Fail(t, "should not run this")
				return &Logger{Name: "root"}
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("use Decorate in root", func(t *testing.T) {
		redis := fx.Module("redis",
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "decorated logger", l.Name)
			}),
		)
		logger := fx.Module("logger",
			fx.Provide(func() *Logger {
				return &Logger{Name: "logger"}
			}),
		)
		app := fxtest.New(t,
			redis,
			logger,
			fx.Decorate(func(l *Logger) *Logger {
				return &Logger{Name: "decorated " + l.Name}
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("use Decorate with Annotate", func(t *testing.T) {
		type Coffee struct {
			Name  string
			Price int
		}

		cafe := fx.Module("cafe",
			fx.Provide(fx.Annotate(func() *Coffee {
				return &Coffee{Name: "Americano", Price: 3}
			}, fx.ResultTags(`group:"coffee"`))),
			fx.Provide(fx.Annotate(func() *Coffee {
				return &Coffee{Name: "Cappucino", Price: 4}
			}, fx.ResultTags(`group:"coffee"`))),
			fx.Provide(fx.Annotate(func() *Coffee {
				return &Coffee{Name: "Cold Brew", Price: 4}
			}, fx.ResultTags(`group:"coffee"`))),
		)

		takeout := fx.Module("takeout",
			cafe,
			fx.Decorate(fx.Annotate(func(coffee []*Coffee) []*Coffee {
				var newC []*Coffee
				for _, c := range coffee {
					newC = append(newC, &Coffee{
						Name:  c.Name,
						Price: c.Price + 1,
					})
				}
				return newC
			}, fx.ParamTags(`group:"coffee"`), fx.ResultTags(`group:"coffee"`))),
			fx.Invoke(fx.Annotate(func(coffee []*Coffee) {
				assert.Equal(t, 3, len(coffee))
				totalPrice := 0
				for _, c := range coffee {
					totalPrice += c.Price
				}
				assert.Equal(t, 4+5+5, totalPrice)
			}, fx.ParamTags(`group:"coffee"`))),
		)

		app := fxtest.New(t,
			takeout,
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("use Decorate with parameter/result struct", func(t *testing.T) {
		type Logger struct {
			Name string
		}
		type A struct {
			fx.In

			Log     *Logger
			Version int `name:"versionNum"`
		}
		type B struct {
			fx.Out

			Log     *Logger
			Version int `name:"versionNum"`
		}
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() int { return 1 },
					fx.ResultTags(`name:"versionNum"`)),
				func() *Logger {
					return &Logger{Name: "logger"}
				},
			),
			fx.Decorate(func(a A) B {
				return B{
					Log:     &Logger{Name: a.Log.Name + " decorated"},
					Version: a.Version + 1,
				}
			}),
			fx.Invoke(fx.Annotate(func(l *Logger, ver int) {
				assert.Equal(t, "logger decorated", l.Name)
				assert.Equal(t, 2, ver)
			}, fx.ParamTags(``, `name:"versionNum"`))),
		)
		defer app.RequireStart().RequireStop()
	})
}
