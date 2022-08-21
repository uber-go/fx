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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/fx/internal/fxlog"
)

func TestModuleSuccess(t *testing.T) {
	t.Parallel()

	type Logger struct {
		Name string
	}

	t.Run("provide a dependency from a submodule", func(t *testing.T) {
		t.Parallel()

		redis := fx.Module("redis",
			fx.Provide(func() *Logger {
				return &Logger{Name: "redis"}
			}),
		)

		app := fxtest.New(t,
			redis,
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "redis", l.Name)
			}),
		)

		defer app.RequireStart().RequireStop()
	})

	t.Run("provide a dependency from nested modules", func(t *testing.T) {
		t.Parallel()
		app := fxtest.New(t,
			fx.Module("child",
				fx.Module("grandchild",
					fx.Provide(func() *Logger {
						return &Logger{Name: "redis"}
					}),
				),
			),
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "redis", l.Name)
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("provide a value to a soft group value from nested modules", func(t *testing.T) {
		t.Parallel()
		type Param struct {
			fx.In

			Foos []string `group:"foo,soft"`
			Bar  int
		}
		type Result struct {
			fx.Out

			Foo string `group:"foo"`
			Bar int
		}
		app := fxtest.New(t,
			fx.Module("child",
				fx.Module("grandchild",
					fx.Provide(fx.Annotate(
						func() string {
							require.FailNow(t, "should not be called")
							return "there"
						},
						fx.ResultTags(`group:"foo"`),
					)),
					fx.Provide(func() Result {
						return Result{Foo: "hello", Bar: 10}
					}),
				),
			),
			fx.Invoke(func(p Param) {
				assert.ElementsMatch(t, []string{"hello"}, p.Foos)
			}),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("invoke from nested module", func(t *testing.T) {
		t.Parallel()
		invokeRan := false
		app := fxtest.New(t,
			fx.Provide(func() *Logger {
				return &Logger{
					Name: "redis",
				}
			}),
			fx.Module("child",
				fx.Module("grandchild",
					fx.Invoke(func(l *Logger) {
						assert.Equal(t, "redis", l.Name)
						invokeRan = true
					}),
				),
			),
		)
		require.True(t, invokeRan)
		require.NoError(t, app.Err())
		defer app.RequireStart().RequireStop()
	})

	t.Run("invoke in module with dep from top module", func(t *testing.T) {
		t.Parallel()
		child := fx.Module("child",
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, "my logger", l.Name)
			}),
		)
		app := fxtest.New(t,
			child,
			fx.Provide(func() *Logger {
				return &Logger{Name: "my logger"}
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("provide in module with annotate", func(t *testing.T) {
		t.Parallel()
		child := fx.Module("child",
			fx.Provide(fx.Annotate(func() *Logger {
				return &Logger{Name: "good logger"}
			}, fx.ResultTags(`name:"goodLogger"`))),
		)
		app := fxtest.New(t,
			child,
			fx.Invoke(fx.Annotate(func(l *Logger) {
				assert.Equal(t, "good logger", l.Name)
			}, fx.ParamTags(`name:"goodLogger"`))),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("invoke in module with annotate", func(t *testing.T) {
		t.Parallel()
		ranInvoke := false
		child := fx.Module("child",
			// use something provided by the root module.
			fx.Invoke(fx.Annotate(func(l *Logger) {
				assert.Equal(t, "good logger", l.Name)
				ranInvoke = true
			})),
		)
		app := fxtest.New(t,
			child,
			fx.Provide(fx.Annotate(func() *Logger {
				return &Logger{Name: "good logger"}
			})),
		)
		defer app.RequireStart().RequireStop()
		assert.True(t, ranInvoke)
	})

	t.Run("use Options in Module", func(t *testing.T) {
		t.Parallel()

		opts := fx.Options(
			fx.Provide(fx.Annotate(func() string {
				return "dog"
			}, fx.ResultTags(`group:"pets"`))),
			fx.Provide(fx.Annotate(func() string {
				return "cat"
			}, fx.ResultTags(`group:"pets"`))),
		)

		petModule := fx.Module("pets", opts)

		app := fxtest.New(t,
			petModule,
			fx.Invoke(fx.Annotate(func(pets []string) {
				assert.ElementsMatch(t, []string{"dog", "cat"}, pets)
			}, fx.ParamTags(`group:"pets"`))),
		)

		defer app.RequireStart().RequireStop()
	})

	t.Run("Invoke order in Modules", func(t *testing.T) {
		t.Parallel()

		type person struct {
			age int
		}

		app := fxtest.New(t,
			fx.Provide(func() *person {
				return &person{
					age: 1,
				}
			}),
			fx.Invoke(func(p *person) {
				assert.Equal(t, 2, p.age)
				p.age += 1
			}),
			fx.Module("module",
				fx.Invoke(func(p *person) {
					assert.Equal(t, 1, p.age)
					p.age += 1
				}),
			),
			fx.Invoke(func(p *person) {
				assert.Equal(t, 3, p.age)
			}),
		)
		require.NoError(t, app.Err())
	})
}

func TestModuleFailures(t *testing.T) {
	t.Parallel()

	t.Run("invoke from submodule failed", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		sub := fx.Module("sub",
			fx.Provide(func() *A { return &A{} }),
			fx.Invoke(func(*A, *B) { // missing dependency.
				require.Fail(t, "this should not be called")
			}),
		)

		app := NewForTest(t,
			sub,
			fx.Invoke(func(a *A) {
				assert.NotNil(t, a)
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing type: *fx_test.B")
	})

	t.Run("provide the same dependency from multiple modules", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		app := NewForTest(t,
			fx.Module("mod1", fx.Provide(func() A { return A{} })),
			fx.Module("mod2", fx.Provide(func() A { return A{} })),
			fx.Invoke(func(a A) {}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already provided by ")
	})

	t.Run("providing Modules should fail", func(t *testing.T) {
		t.Parallel()
		app := NewForTest(t,
			fx.Module("module",
				fx.Provide(
					fx.Module("module"),
				),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Provide")
	})

	t.Run("invoking Modules should fail", func(t *testing.T) {
		t.Parallel()
		app := NewForTest(t,
			fx.Module("module",
				fx.Invoke(
					fx.Invoke("module"),
				),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Invoke")
	})

	t.Run("annotate failure in Module", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		newA := func() A {
			return A{}
		}

		app := NewForTest(t,
			fx.Module("module",
				fx.Provide(fx.Annotate(newA,
					fx.ParamTags(`"name:"A1"`),
					fx.ParamTags(`"name:"A2"`),
				)),
			),
		)
		err := app.Err()
		require.Error(t, err)

		assert.Contains(t, err.Error(), "encountered error while applying annotation")
		assert.Contains(t, err.Error(), "cannot apply more than one line of ParamTags")
	})

	t.Run("soft provided to fx.Out struct", func(t *testing.T) {
		t.Parallel()

		type Result struct {
			fx.Out

			Bars []int `group:"bar,soft"`
		}
		app := NewForTest(t,
			fx.Provide(func() Result { return Result{Bars: []int{1, 2, 3}} }),
		)
		err := app.Err()
		require.Error(t, err, "failed to create app")
		assert.Contains(t, err.Error(), "cannot use soft with result value groups")
	})

	t.Run("provider in Module fails", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		newA := func() (A, error) {
			return A{}, nil
		}
		newB := func() (B, error) {
			return B{}, errors.New("minor sadness")
		}

		app := NewForTest(t,
			fx.Module("module",
				fx.Provide(
					newA,
					newB,
				),
			),
			fx.Invoke(func(A, B) {
				assert.Fail(t, "this should never run")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build fx_test.B")
		assert.Contains(t, err.Error(), "minor sadness")
	})

	t.Run("invalid Options in Module", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			desc string
			opt  fx.Option
		}{
			{
				desc: "StartTimeout Option",
				opt:  fx.StartTimeout(time.Second),
			},
			{
				desc: "StopTimeout Option",
				opt:  fx.StopTimeout(time.Second),
			},
			{
				desc: "WithLogger Option",
				opt:  fx.WithLogger(func() fxevent.Logger { return new(fxlog.Spy) }),
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()

				app := NewForTest(t,
					fx.Module("module",
						tt.opt,
					),
				)
				require.Error(t, app.Err())
			})
		}
	})
}
