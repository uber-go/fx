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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestDecorateSuccess(t *testing.T) {
	type Logger struct {
		Name string
	}

	t.Run("objects provided by other modules are decorated", func(t *testing.T) {
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

	t.Run("objects in child modules are decorated.", func(t *testing.T) {
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

	t.Run("root decoration applies to all modules", func(t *testing.T) {
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

	t.Run("decorator with soft value group", func(t *testing.T) {
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() (string, int) { return "cheeseburger", 15 },
					fx.ResultTags(`group:"burger"`, `group:"potato"`),
				),
			),
			fx.Provide(
				fx.Annotate(
					func() (string, int) { return "mushroomburger", 35 },
					fx.ResultTags(`group:"burger"`, `group:"potato"`),
				),
			),
			fx.Provide(
				fx.Annotate(
					func() string {
						require.FailNow(t, "should not be called")
						return "veggieburger"
					},
					fx.ResultTags(`group:"burger"`, `group:"potato"`),
				),
			),
			fx.Decorate(
				fx.Annotate(
					func(burgers []string) []string {
						retBurg := make([]string, len(burgers))
						for i, burger := range burgers {
							retBurg[i] = strings.ToUpper(burger)
						}
						return retBurg
					},
					fx.ParamTags(`group:"burger,soft"`),
					fx.ResultTags(`group:"burger"`),
				),
			),
			fx.Invoke(
				fx.Annotate(
					func(burgers []string, fries []int) {
						assert.ElementsMatch(t, []string{"CHEESEBURGER", "MUSHROOMBURGER"}, burgers)
					},
					fx.ParamTags(`group:"burger,soft"`, `group:"potato"`),
				),
			),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("decorator with optional parameter", func(t *testing.T) {
		type Config struct {
			Name string
		}
		type Logger struct {
			Name string
		}
		type DecoratorParam struct {
			fx.In

			Cfg *Config `optional:"true"`
			Log *Logger
		}

		app := fxtest.New(t,
			fx.Provide(func() *Logger { return &Logger{Name: "log"} }),
			fx.Decorate(func(p DecoratorParam) *Logger {
				if p.Cfg != nil {
					return &Logger{Name: p.Cfg.Name}
				}
				return &Logger{Name: p.Log.Name}
			}),
			fx.Invoke(func(l *Logger) {
				assert.Equal(t, l.Name, "log")
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("transitive decoration", func(t *testing.T) {
		type Config struct {
			Scope string
		}
		type Logger struct {
			Cfg *Config
		}
		app := fxtest.New(t,
			fx.Provide(func() *Config { return &Config{Scope: "root"} }),
			fx.Module("child",
				fx.Decorate(func() *Config { return &Config{Scope: "child"} }),
				fx.Provide(func(cfg *Config) *Logger { return &Logger{Cfg: cfg} }),
				fx.Invoke(func(l *Logger) {
					assert.Equal(t, "child", l.Cfg.Scope)
				}),
			),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("ineffective transitive decoration", func(t *testing.T) {
		type Config struct {
			Scope string
		}
		type Logger struct {
			Cfg *Config
		}
		app := fxtest.New(t,
			fx.Provide(func() *Config {
				return &Config{Scope: "root"}
			}),
			fx.Provide(func(cfg *Config) *Logger {
				return &Logger{Cfg: &Config{
					Scope: cfg.Scope + " logger",
				}}
			}),
			fx.Module("child",
				fx.Decorate(func() *Config {
					return &Config{Scope: "child"}
				}),
				// Logger does not get replaced since it was provided
				// from a different Scope.
				fx.Invoke(func(l *Logger) {
					assert.Equal(t, "root logger", l.Cfg.Scope)
				}),
			),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("decoration must execute when required by a member of group", func(t *testing.T) {
		type Drinks interface {
		}
		type Coffee struct {
			Type  string
			Name  string
			Price int
		}
		type PriceService struct {
			DefaultPrice int
		}
		app := fxtest.New(t,
			fx.Provide(func() *PriceService {
				return &PriceService{DefaultPrice: 3}
			}),
			fx.Decorate(func(service *PriceService) *PriceService {
				service.DefaultPrice = 10
				return service
			}),
			fx.Provide(fx.Annotate(func(service *PriceService) Drinks {
				assert.Equal(t, 10, service.DefaultPrice)
				return &Coffee{Type: "coffee", Name: "Americano", Price: service.DefaultPrice}
			}, fx.ResultTags(`group:"drinks"`))),
			fx.Provide(fx.Annotated{Group: "drinks", Target: func() Drinks {
				return &Coffee{Type: "coffee", Name: "Cold Brew", Price: 4}
			}}),
			fx.Invoke(fx.Annotate(func(drinks []Drinks) {
				assert.Len(t, drinks, 2)
			}, fx.ParamTags(`group:"drinks"`))),
		)
		defer app.RequireStart().RequireStop()
	})
}

func TestDecorateFailure(t *testing.T) {
	t.Run("decorator returns an error", func(t *testing.T) {
		type Logger struct {
			Name string
		}

		app := NewForTest(t,
			fx.Provide(func() *Logger {
				return &Logger{Name: "root"}
			}),
			fx.Decorate(func(l *Logger) (*Logger, error) {
				return &Logger{Name: l.Name + "decorated"}, errors.New("minor sadness")
			}),
			fx.Invoke(func(l *Logger) {
				assert.Fail(t, "this should not be executed")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minor sadness")
	})

	t.Run("decorator in a nested module returns an error", func(t *testing.T) {
		type Logger struct {
			Name string
		}

		app := NewForTest(t,
			fx.Provide(func() *Logger {
				return &Logger{Name: "root"}
			}),
			fx.Module("child",
				fx.Decorate(func(l *Logger) *Logger {
					return &Logger{Name: l.Name + "decorated"}
				}),
				fx.Decorate(func(l *Logger) *Logger {
					return &Logger{Name: l.Name + "decorated"}
				}),
				fx.Invoke(func(l *Logger) {
					assert.Fail(t, "this should not be executed")
				}),
			),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "*fx_test.Logger already decorated")
	})

	t.Run("decorating a type more than once in the same Module errors", func(t *testing.T) {
		type Logger struct {
			Name string
		}

		app := NewForTest(t,
			fx.Provide(func() *Logger {
				return &Logger{Name: "root"}
			}),
			fx.Decorate(func(l *Logger) *Logger {
				return &Logger{Name: "dec1 " + l.Name}
			}),
			fx.Decorate(func(l *Logger) *Logger {
				return &Logger{Name: "dec2 " + l.Name}
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "*fx_test.Logger already decorated")
	})

	t.Run("annotated decorator returns an error", func(t *testing.T) {
		type Logger struct {
			Name string
		}

		tag := `name:"decoratedLogger"`
		app := NewForTest(t,
			fx.Provide(fx.Annotate(func() *Logger {
				return &Logger{Name: "root"}
			}, fx.ResultTags(tag))),
			fx.Decorate(fx.Annotate(func(l *Logger) (*Logger, error) {
				return &Logger{Name: "dec1 " + l.Name}, errors.New("major sadness")
			}, fx.ParamTags(tag), fx.ResultTags(tag))),
			fx.Invoke(fx.Annotate(func(l *Logger) {
				assert.Fail(t, "this should never run")
			}, fx.ParamTags(tag))),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "major sadness")
	})

	t.Run("all decorator dependencies must be provided", func(t *testing.T) {
		type Logger struct {
			Name string
		}
		type Config struct {
			Name string
		}

		app := NewForTest(t,
			fx.Provide(func() *Logger {
				return &Logger{Name: "logger"}
			}),
			fx.Decorate(func(l *Logger, c *Config) *Logger {
				return &Logger{Name: l.Name + c.Name}
			}),
			fx.Invoke(func(l *Logger) {
				assert.Fail(t, "this should never run")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing dependencies")
	})

	t.Run("decorate cannot provide a non-existent type", func(t *testing.T) {
		type Logger struct {
			Name string
		}

		app := NewForTest(t,
			fx.Decorate(func() *Logger {
				return &Logger{Name: "decorator"}
			}),
			fx.Invoke(func(l *Logger) {
				assert.Fail(t, "this should never run")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing dependencies")
	})
}
