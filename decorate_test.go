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
	"context"
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

	t.Run("decorator with soft map group", func(t *testing.T) {
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() (string, int) { return "cheeseburger", 15 },
					fx.ResultTags(`name:"cheese" group:"burger"`, `name:"cheese" group:"potato"`),
				),
			),
			fx.Provide(
				fx.Annotate(
					func() (string, int) { return "mushroomburger", 35 },
					fx.ResultTags(`name:"mushroom" group:"burger"`, `name:"mushroom" group:"potato"`),
				),
			),
			fx.Provide(
				fx.Annotate(
					func() string {
						require.FailNow(t, "should not be called")
						return "veggieburger"
					},
					fx.ResultTags(`name:"veggie" group:"burger"`, `name:"veggie" group:"potato"`),
				),
			),
			fx.Decorate(
				fx.Annotate(
					func(burgers map[string]string) map[string]string {
						retBurg := make(map[string]string)
						for key, burger := range burgers {
							retBurg[key] = strings.ToUpper(burger)
						}
						return retBurg
					},
					fx.ParamTags(`group:"burger,soft"`),
					fx.ResultTags(`group:"burger"`),
				),
			),
			fx.Invoke(
				fx.Annotate(
					func(burgers map[string]string, fries map[string]int) {
						expected := map[string]string{"cheese": "CHEESEBURGER", "mushroom": "MUSHROOMBURGER"}
						assert.Equal(t, expected, burgers)
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
		type Drinks any
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

	t.Run("slice decorators are blocked for named value groups", func(t *testing.T) {
		// This test verifies the key design decision from dig PR #381:
		// Slice decorators cannot be used with named value groups because
		// they would break the map functionality

		type Service struct {
			Name string
		}

		type DecorationInput struct {
			fx.In
			// Try to consume as slice for decoration
			Services []Service `group:"services"`
		}

		type DecorationOutput struct {
			fx.Out
			// Output as slice - this should break map consumption
			Services []Service `group:"services"`
		}

		sliceDecorator := func(input DecorationInput) DecorationOutput {
			// This slice decorator executes due to current dig limitation
			// but consumption will fail with proper validation error
			enhanced := make([]Service, len(input.Services))
			for i, service := range input.Services {
				enhanced[i] = Service{Name: "[DECORATED]" + service.Name}
			}
			return DecorationOutput{Services: enhanced}
		}

		app := NewForTest(t,
			fx.Provide(
				// Provide with names (making this a named value group)
				fx.Annotate(
					func() Service { return Service{Name: "auth"} },
					fx.ResultTags(`name:"auth" group:"services"`),
				),
				fx.Annotate(
					func() Service { return Service{Name: "billing"} },
					fx.ResultTags(`name:"billing" group:"services"`),
				),
			),
			// This slice decorator should be blocked for named value groups
			fx.Decorate(sliceDecorator),
			// Try to consume as slice - this should trigger the dig validation
			fx.Invoke(fx.Annotate(
				func(serviceSlice []Service) {
					t.Logf("ServiceSlice length: %d", len(serviceSlice))
				},
				fx.ParamTags(`group:"services"`),
			)),
		)

		// Decoration fails at invoke/start time, not decorate time
		err := app.Start(context.Background())
		defer app.Stop(context.Background())

		// Should ALWAYS fail with the specific dig validation error
		require.Error(t, err, "Slice consumption after slice decoration of named groups should fail")

		// Should get the specific dig error about slice decoration
		assert.Contains(t, err.Error(), "cannot use slice decoration for value group",
			"Expected dig slice decoration error, got: %v", err)
		assert.Contains(t, err.Error(), "group contains named values",
			"Expected error about named values, got: %v", err)
		assert.Contains(t, err.Error(), "use map[string]T decorator instead",
			"Expected suggestion to use map decorator, got: %v", err)
	})

	t.Run("map decorators work fine with named value groups", func(t *testing.T) {
		// This test shows the contrast - map decorators work perfectly
		// with named value groups, unlike slice decorators which break them

		type Service struct {
			Name string
		}

		type DecorationInput struct {
			fx.In
			// Consume as map for decoration - this works fine
			Services map[string]Service `group:"services"`
		}

		type DecorationOutput struct {
			fx.Out
			// Output as map - preserves the name-to-value mapping
			Services map[string]Service `group:"services"`
		}

		mapDecorator := func(input DecorationInput) DecorationOutput {
			// This decorator preserves the map structure and names
			enhanced := make(map[string]Service)
			for name, service := range input.Services {
				enhanced[name] = Service{Name: "[MAP_DECORATED]" + service.Name}
			}
			return DecorationOutput{Services: enhanced}
		}

		type FinalParams struct {
			fx.In
			// Consume as map after map decoration - this should work perfectly
			ServiceMap map[string]Service `group:"services"`
		}

		var params FinalParams
		app := NewForTest(t,
			fx.Provide(
				// Provide with names (making this a named value group)
				fx.Annotate(
					func() Service { return Service{Name: "auth"} },
					fx.ResultTags(`name:"auth" group:"services"`),
				),
				fx.Annotate(
					func() Service { return Service{Name: "billing"} },
					fx.ResultTags(`name:"billing" group:"services"`),
				),
			),
			// This map decorator should work fine with named value groups
			fx.Decorate(mapDecorator),
			fx.Populate(&params),
		)

		// Should succeed - map decoration preserves map functionality
		err := app.Start(context.Background())
		defer app.Stop(context.Background())

		require.NoError(t, err, "Map decoration should work fine with named value groups")

		// Verify the final populated params also work correctly
		require.Len(t, params.ServiceMap, 2)
		assert.Equal(t, "[MAP_DECORATED]auth", params.ServiceMap["auth"].Name)
		assert.Equal(t, "[MAP_DECORATED]billing", params.ServiceMap["billing"].Name)
	})
}

// Test processor types for map decoration tests
type testProcessor interface {
	Process(input string) string
	Name() string
}

type testBasicProcessor struct {
	name string
}

func (b *testBasicProcessor) Process(input string) string {
	return b.name + ": " + input
}

func (b *testBasicProcessor) Name() string {
	return b.name
}

type testEnhancedProcessor struct {
	wrapped testProcessor
	prefix  string
}

func (e *testEnhancedProcessor) Process(input string) string {
	return e.prefix + " " + e.wrapped.Process(input)
}

func (e *testEnhancedProcessor) Name() string {
	return e.wrapped.Name()
}

// TestMapValueGroupsDecoration tests decoration of map value groups
func TestMapValueGroupsDecoration(t *testing.T) {
	t.Parallel()

	t.Run("decorate map value groups", func(t *testing.T) {
		t.Parallel()

		type DecorationInput struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		type DecorationOutput struct {
			fx.Out
			Processors map[string]testProcessor `group:"processors"`
		}

		decorateProcessors := func(input DecorationInput) DecorationOutput {
			enhanced := make(map[string]testProcessor)
			for name, processor := range input.Processors {
				enhanced[name] = &testEnhancedProcessor{
					wrapped: processor,
					prefix:  "[ENHANCED]",
				}
			}
			return DecorationOutput{Processors: enhanced}
		}

		type FinalParams struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		var params FinalParams
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "json"} },
					fx.ResultTags(`name:"json" group:"processors"`),
				),
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "xml"} },
					fx.ResultTags(`name:"xml" group:"processors"`),
				),
			),
			fx.Decorate(decorateProcessors),
			fx.Populate(&params),
		)

		err := app.Start(context.Background())
		defer app.Stop(context.Background())
		require.NoError(t, err)

		require.Len(t, params.Processors, 2)

		// Test that processors are decorated
		jsonResult := params.Processors["json"].Process("data")
		assert.Equal(t, "[ENHANCED] json: data", jsonResult)

		xmlResult := params.Processors["xml"].Process("data")
		assert.Equal(t, "[ENHANCED] xml: data", xmlResult)

		// Names should be preserved
		assert.Equal(t, "json", params.Processors["json"].Name())
		assert.Equal(t, "xml", params.Processors["xml"].Name())
	})

	t.Run("single decoration layer", func(t *testing.T) {
		t.Parallel()

		type DecorationInput struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		type DecorationOutput struct {
			fx.Out
			Processors map[string]testProcessor `group:"processors"`
		}

		decoration := func(input DecorationInput) DecorationOutput {
			enhanced := make(map[string]testProcessor)
			for name, processor := range input.Processors {
				enhanced[name] = &testEnhancedProcessor{
					wrapped: processor,
					prefix:  "[DECORATED]",
				}
			}
			return DecorationOutput{Processors: enhanced}
		}

		type FinalParams struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		var params FinalParams
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "base"} },
					fx.ResultTags(`name:"base" group:"processors"`),
				),
			),
			fx.Decorate(decoration),
			fx.Populate(&params),
		)

		err := app.Start(context.Background())
		defer app.Stop(context.Background())
		require.NoError(t, err)

		require.Len(t, params.Processors, 1)

		// Test decoration
		result := params.Processors["base"].Process("test")
		assert.Equal(t, "[DECORATED] base: test", result)
	})

	t.Run("decoration preserves map keys", func(t *testing.T) {
		t.Parallel()

		type DecorationInput struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		type DecorationOutput struct {
			fx.Out
			Processors map[string]testProcessor `group:"processors"`
		}

		var decorationInputKeys []string
		var decorationOutputKeys []string

		decorateWithKeyTracking := func(input DecorationInput) DecorationOutput {
			decorationInputKeys = make([]string, 0, len(input.Processors))
			for key := range input.Processors {
				decorationInputKeys = append(decorationInputKeys, key)
			}

			enhanced := make(map[string]testProcessor)
			for name, processor := range input.Processors {
				enhanced[name] = &testEnhancedProcessor{
					wrapped: processor,
					prefix:  "[TRACKED]",
				}
			}

			decorationOutputKeys = make([]string, 0, len(enhanced))
			for key := range enhanced {
				decorationOutputKeys = append(decorationOutputKeys, key)
			}

			return DecorationOutput{Processors: enhanced}
		}

		type FinalParams struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		var params FinalParams
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "alpha"} },
					fx.ResultTags(`name:"alpha" group:"processors"`),
				),
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "beta"} },
					fx.ResultTags(`name:"beta" group:"processors"`),
				),
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "gamma"} },
					fx.ResultTags(`name:"gamma" group:"processors"`),
				),
			),
			fx.Decorate(decorateWithKeyTracking),
			fx.Populate(&params),
		)

		err := app.Start(context.Background())
		defer app.Stop(context.Background())
		require.NoError(t, err)

		require.Len(t, params.Processors, 3)

		// Verify keys are preserved through decoration
		assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, decorationInputKeys)
		assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, decorationOutputKeys)

		// Verify final map has correct keys
		finalKeys := make([]string, 0, len(params.Processors))
		for key := range params.Processors {
			finalKeys = append(finalKeys, key)
		}
		assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, finalKeys)
	})

	t.Run("map decoration across modules", func(t *testing.T) {
		t.Parallel()

		type DecorationInput struct {
			fx.In
			Processors map[string]testProcessor `group:"processors"`
		}

		type DecorationOutput struct {
			fx.Out
			Processors map[string]testProcessor `group:"processors"`
		}

		var outerProcessors map[string]testProcessor
		var innerProcessors map[string]testProcessor

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "auth"} },
					fx.ResultTags(`name:"auth" group:"processors"`),
				),
				fx.Annotate(
					func() testProcessor { return &testBasicProcessor{name: "billing"} },
					fx.ResultTags(`name:"billing" group:"processors"`),
				),
			),
			fx.Decorate(func(input DecorationInput) DecorationOutput {
				enhanced := make(map[string]testProcessor)
				for name, processor := range input.Processors {
					enhanced[name] = &testEnhancedProcessor{
						wrapped: processor,
						prefix:  "[OUTER]",
					}
				}
				return DecorationOutput{Processors: enhanced}
			}),
			fx.Invoke(fx.Annotate(
				func(processors map[string]testProcessor) {
					outerProcessors = processors
				},
				fx.ParamTags(`group:"processors"`),
			)),
			fx.Module("mymodule",
				fx.Decorate(func(input DecorationInput) DecorationOutput {
					enhanced := make(map[string]testProcessor)
					for name, processor := range input.Processors {
						enhanced[name] = &testEnhancedProcessor{
							wrapped: processor,
							prefix:  "[INNER]",
						}
					}
					return DecorationOutput{Processors: enhanced}
				}),
				fx.Invoke(fx.Annotate(
					func(processors map[string]testProcessor) {
						innerProcessors = processors
					},
					fx.ParamTags(`group:"processors"`),
				)),
			),
		)
		defer app.RequireStart().RequireStop()

		t.Logf("Outer processors:")
		for name, p := range outerProcessors {
			t.Logf("  %s: %s", name, p.Process("data"))
		}
		t.Logf("Inner processors:")
		for name, p := range innerProcessors {
			t.Logf("  %s: %s", name, p.Process("data"))
		}

		// Test that map decoration chains across modules
		require.Len(t, outerProcessors, 2)
		assert.Equal(t, "[OUTER] auth: data", outerProcessors["auth"].Process("data"))
		assert.Equal(t, "[OUTER] billing: data", outerProcessors["billing"].Process("data"))

		require.Len(t, innerProcessors, 2)
		assert.Equal(t, "[INNER] [OUTER] auth: data", innerProcessors["auth"].Process("data"))
		assert.Equal(t, "[INNER] [OUTER] billing: data", innerProcessors["billing"].Process("data"))
	})
}
