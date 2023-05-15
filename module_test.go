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
	"bytes"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/zap"
)

func TestModuleSuccess(t *testing.T) {
	t.Parallel()

	type Logger struct {
		Name string
	}

	type Foo struct {
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

	t.Run("custom logger for module", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			desc           string
			giveWithLogger fx.Option
			wantEvents     []string
		}{
			{
				desc:           "custom logger for module",
				giveWithLogger: fx.NopLogger,
				wantEvents: []string{
					"Supplied", "Provided", "Provided", "Provided",
					"Run", "LoggerInitialized", "Invoking", "Invoked",
				},
			},
			{
				desc:           "Not using a custom logger for module defaults to app logger",
				giveWithLogger: fx.Options(),
				wantEvents: []string{
					"Supplied", "Provided", "Provided", "Provided", "Provided", "Run",
					"LoggerInitialized", "Invoking", "Run", "Invoked", "Invoking", "Invoked",
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				var spy fxlog.Spy

				redis := fx.Module("redis",
					fx.Provide(func() *Foo {
						return &Foo{Name: "redis"}
					}),
					fx.Invoke(func(r *Foo) {
						assert.Equal(t, "redis", r.Name)
					}),
					tt.giveWithLogger,
				)

				app := fxtest.New(t,
					redis,
					fx.Supply(&spy),
					fx.WithLogger(func(spy *fxlog.Spy) fxevent.Logger {
						return spy
					}),
					fx.Invoke(func(r *Foo) {
						assert.Equal(t, "redis", r.Name)
					}),
				)

				// events from module with a custom logger not logged in app logger
				assert.Equal(t, tt.wantEvents, spy.EventTypes())

				spy.Reset()
				app.RequireStart().RequireStop()

				require.NoError(t, app.Err())

				assert.Equal(t, []string{"Started", "Stopped"}, spy.EventTypes())
			})
		}
	})

	type NamedSpy struct {
		fxlog.Spy
		name string
	}

	t.Run("decorator on module logger does not affect app logger", func(t *testing.T) {
		t.Parallel()

		appSpy := NamedSpy{name: "app"}
		moduleSpy := NamedSpy{name: "redis"}

		redis := fx.Module("redis",
			fx.Provide(func() *Foo {
				return &Foo{Name: "redis"}
			}),
			fx.Supply(&appSpy),
			fx.Replace(&moduleSpy),
			fx.WithLogger(func(spy *NamedSpy) fxevent.Logger {
				assert.Equal(t, "redis", spy.name)
				return spy
			}),
			fx.Invoke(func(r *Foo) {
				assert.Equal(t, "redis", r.Name)
			}),
		)

		app := fxtest.New(t,
			redis,
			fx.WithLogger(func(spy *NamedSpy) fxevent.Logger {
				assert.Equal(t, "app", spy.name)
				return spy
			}),
			fx.Invoke(func(r *Foo) {
				assert.Equal(t, "redis", r.Name)
			}),
		)

		assert.Equal(t, []string{
			"Provided", "Supplied", "Replaced", "Run", "Run",
			"LoggerInitialized", "Invoking", "Run", "Invoked",
		}, moduleSpy.EventTypes())

		assert.Equal(t, []string{
			"Provided", "Provided", "Provided",
			"LoggerInitialized", "Invoking", "Invoked",
		}, appSpy.EventTypes())

		appSpy.Reset()
		moduleSpy.Reset()

		app.RequireStart().RequireStop()

		require.NoError(t, app.Err())

		assert.Equal(t, []string{"Started", "Stopped"}, appSpy.EventTypes())
		assert.Empty(t, moduleSpy.EventTypes())
	})

	t.Run("module uses parent module's logger to log events", func(t *testing.T) {
		t.Parallel()

		appSpy := NamedSpy{name: "app"}
		childSpy := NamedSpy{name: "child"}

		grandchild := fx.Module("grandchild",
			fx.Provide(func() *Foo {
				return &Foo{Name: "grandchild"}
			}),
			fx.Invoke(func(r *Foo) {
				assert.Equal(t, "grandchild", r.Name)
			}),
		)

		child := fx.Module("child",
			grandchild,
			fx.Supply(&appSpy),
			fx.Replace(&childSpy),
			fx.WithLogger(func(spy *NamedSpy) fxevent.Logger {
				assert.Equal(t, "child", spy.name)
				return spy
			}),
			fx.Invoke(func(r *Foo) {
				assert.Equal(t, "grandchild", r.Name)
			}),
		)

		app := fxtest.New(t,
			child,
			fx.WithLogger(func(spy *NamedSpy) fxevent.Logger {
				assert.Equal(t, "app", spy.name)
				return spy
			}),
			fx.Invoke(func(r *Foo) {
				assert.Equal(t, "grandchild", r.Name)
			}),
		)

		assert.Equal(t, []string{
			"Supplied", "Provided", "Replaced", "Run", "Run", "LoggerInitialized",
			//Invoke logged twice, once from child and another from grandchild
			"Invoking", "Run", "Invoked", "Invoking", "Invoked",
		}, childSpy.EventTypes(), "events from grandchild also logged in child logger")

		assert.Equal(t, []string{
			"Provided", "Provided", "Provided",
			"LoggerInitialized", "Invoking", "Invoked",
		}, appSpy.EventTypes(), "events from modules do not appear in app logger")

		appSpy.Reset()
		childSpy.Reset()

		app.RequireStart().RequireStop()

		require.NoError(t, app.Err())

		assert.Equal(t, []string{"Started", "Stopped"}, appSpy.EventTypes())
		assert.Empty(t, childSpy.EventTypes())
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
					fx.ParamTags(`name:"A1"`),
					fx.ParamTags(`name:"A2"`),
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
				desc: "Logger Option",
				opt:  fx.Logger(log.New(&bytes.Buffer{}, "", 0)),
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

	t.Run("invalid WithLogger in Module", func(t *testing.T) {
		t.Parallel()

		var spy fxlog.Spy

		spyAsLogger := fx.Options(
			fx.Supply(&spy),
			fx.WithLogger(func(spy *fxlog.Spy) fxevent.Logger {
				return spy
			}),
		)

		defaultModuleOpts := fx.Options(
			fx.Provide(func() string {
				return "redis"
			}),
			fx.Invoke(func(s string) {
				assert.Fail(t, "this should never run")
			}),
		)

		tests := []struct {
			desc            string
			giveModuleOpts  fx.Option
			giveAppOpts     fx.Option
			wantErrContains []string
			wantEvents      []string
		}{
			{
				desc:           "error in Provide shows logs in module",
				giveModuleOpts: fx.Options(spyAsLogger, fx.Provide(&bytes.Buffer{})), // not passing in a constructor
				giveAppOpts:    fx.Options(),
				wantErrContains: []string{
					"must provide constructor function, got  (type *bytes.Buffer)",
				},
				wantEvents: []string{
					"Supplied", "Provided", "Run", "LoggerInitialized",
				},
			},
			{
				desc: "logger in module failed to build",
				giveModuleOpts: fx.WithLogger(func() (fxevent.Logger, error) {
					return nil, errors.New("error building logger")
				}),
				giveAppOpts:     spyAsLogger,
				wantErrContains: []string{"error building logger"},
				wantEvents: []string{
					"Supplied", "Provided", "Provided", "Provided", "Run",
					"LoggerInitialized", "Provided", "LoggerInitialized",
				},
			},
			{
				desc: "logger dependency in module failed to build",
				giveModuleOpts: fx.Options(
					fx.Provide(func() (*zap.Logger, error) {
						return nil, errors.New("error building logger dependency")
					}),
					fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
						t.Errorf("WithLogger must not be called")
						panic("must not be called")
					}),
				),
				giveAppOpts:     spyAsLogger,
				wantErrContains: []string{"error building logger dependency"},
				wantEvents: []string{
					"Supplied", "Provided", "Provided", "Provided", "Run",
					"LoggerInitialized", "Provided", "Provided", "Run", "LoggerInitialized",
				},
			},
			{
				desc:           "Invalid input for WithLogger",
				giveModuleOpts: fx.WithLogger(&fxlog.Spy{}), // not passing in a constructor for WithLogger
				giveAppOpts:    spyAsLogger,
				wantErrContains: []string{
					"fx.WithLogger", "from:", "Failed",
				},
				wantEvents: []string{
					"Supplied", "Provided", "Provided", "Provided", "Run",
					"LoggerInitialized", "Provided", "LoggerInitialized",
				},
			},
		}
		for _, tt := range tests {
			spy.Reset()
			t.Run(tt.desc, func(t *testing.T) {
				redis := fx.Module("redis",
					tt.giveModuleOpts,
					defaultModuleOpts,
				)

				app := fx.New(
					tt.giveAppOpts,
					redis,
					fx.Invoke(func(s string) {
						assert.Fail(t, "this should never run")
					}),
				)
				err := app.Err()
				require.Error(t, err)
				for _, contains := range tt.wantErrContains {
					assert.Contains(t, err.Error(), contains)
				}

				assert.Equal(t,
					tt.wantEvents,
					spy.EventTypes(),
				)
			})
		}
	})
}
