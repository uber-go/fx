// Copyright (c) 2026 Uber Technologies, Inc.
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
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestAnnotateNestedLifecycle(t *testing.T) {
	t.Parallel()

	type hooks struct{ start, stop bool }
	for _, tt := range []struct {
		name      string
		layers    []hooks
		wantStart []string
		wantStop  []string
	}{
		{
			name:      "two layers",
			layers:    []hooks{{true, true}, {true, true}},
			wantStart: []string{"start 0", "start 1"},
			wantStop:  []string{"stop 1", "stop 0"},
		},
		{
			name:      "three mixed layers",
			layers:    []hooks{{true, true}, {false, true}, {true, false}},
			wantStart: []string{"start 0", "start 2"},
			wantStop:  []string{"stop 1", "stop 0"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			var events []string
			var hookValues []*bytes.Buffer
			var target any = func() *bytes.Buffer {
				calls++
				return bytes.NewBufferString("service")
			}
			for i, layer := range tt.layers {
				var annotations []fx.Annotation
				if layer.start {
					annotations = append(annotations, fx.OnStart(func(value *bytes.Buffer) {
						events = append(events, fmt.Sprintf("start %d", i))
						hookValues = append(hookValues, value)
					}))
				}
				if layer.stop {
					annotations = append(annotations, fx.OnStop(func(value *bytes.Buffer) {
						events = append(events, fmt.Sprintf("stop %d", i))
						hookValues = append(hookValues, value)
					}))
				}
				target = fx.Annotate(target, annotations...)
			}
			var value *bytes.Buffer
			app := fxtest.New(t, fx.Provide(target), fx.Populate(&value))
			require.Equal(t, 1, calls)
			require.Empty(t, events)
			app.RequireStart()
			assert.Equal(t, tt.wantStart, events)
			app.RequireStop()
			assert.Equal(t, append(tt.wantStart, tt.wantStop...), events)
			assert.Equal(t, 1, calls)
			for _, hookValue := range hookValues {
				assert.Same(t, value, hookValue)
			}
		})
	}
}

func TestAnnotateNestedInvoke(t *testing.T) {
	t.Parallel()

	var events []string
	app := fxtest.New(t, fx.Invoke(fx.Annotate(
		fx.Annotate(func() { events = append(events, "invoke") },
			fx.OnStart(func() { events = append(events, "inner start") }),
			fx.OnStop(func() { events = append(events, "inner stop") }),
		),
		fx.OnStart(func() { events = append(events, "outer start") }),
		fx.OnStop(func() { events = append(events, "outer stop") }),
	)))
	assert.Equal(t, []string{"invoke"}, events)
	app.RequireStart()
	assert.Equal(t, []string{"invoke", "inner start", "outer start"}, events)
	app.RequireStop()
	assert.Equal(t, []string{"invoke", "inner start", "outer start", "outer stop", "inner stop"}, events)
}

func TestAnnotateNestedAs(t *testing.T) {
	t.Parallel()

	for _, keepSelf := range []bool{false, true} {
		t.Run(fmt.Sprintf("Self=%v", keepSelf), func(t *testing.T) {
			var calls int
			var events []string
			var concrete *bytes.Buffer
			inner := fx.Annotate(func() *bytes.Buffer {
				calls++
				return bytes.NewBufferString("service")
			}, fx.OnStart(func(value *bytes.Buffer) {
				concrete = value
				events = append(events, "inner start")
			}), fx.OnStop(func(value *bytes.Buffer) {
				assert.Same(t, concrete, value)
				events = append(events, "inner stop")
			}))
			annotations := []fx.Annotation{
				fx.As(new(io.Writer)),
				fx.OnStart(func(value io.Writer) {
					assert.Same(t, concrete, value)
					events = append(events, "outer start")
				}),
				fx.OnStop(func(value io.Writer) {
					assert.Same(t, concrete, value)
					events = append(events, "outer stop")
				}),
			}
			if keepSelf {
				annotations = append(annotations, fx.As(fx.Self()))
			}
			target := fx.Annotate(inner, annotations...)
			var writer io.Writer
			var original *bytes.Buffer
			opts := []fx.Option{fx.Provide(target), fx.Populate(&writer)}
			if keepSelf {
				opts = append(opts, fx.Populate(&original))
			}
			app := fxtest.New(t, opts...)
			app.RequireStart().RequireStop()
			assert.Same(t, concrete, writer)
			assert.Equal(t, []string{"inner start", "outer start", "outer stop", "inner stop"}, events)
			if keepSelf {
				assert.Same(t, concrete, original)
			} else {
				err := fx.ValidateApp(fx.NopLogger, fx.Provide(target), fx.Invoke(func(*bytes.Buffer) {}))
				require.ErrorContains(t, err, "missing type")
			}
			assert.Equal(t, 1, calls)
		})
	}
}

func TestAnnotateNestedComposition(t *testing.T) {
	t.Parallel()

	t.Run("parameter and result tags", func(t *testing.T) {
		var result struct {
			fx.In
			Buffer *bytes.Buffer `name:"output"`
		}
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() string { return "tagged" }, fx.ResultTags(`name:"input"`)),
				fx.Annotate(
					fx.Annotate(bytes.NewBufferString, fx.ParamTags(`name:"input"`)),
					fx.ResultTags(`name:"output"`),
				),
			),
			fx.Populate(&result),
		)
		app.RequireStart().RequireStop()
		assert.Equal(t, "tagged", result.Buffer.String())
	})

	t.Run("From with parameter and result tags", func(t *testing.T) {
		var result struct {
			fx.In
			Value string `name:"output"`
		}
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() *bytes.Buffer { return bytes.NewBufferString("reader") }, fx.ResultTags(`name:"input"`)),
				fx.Annotate(
					fx.Annotate(func(reader io.Reader) (string, error) {
						value, err := io.ReadAll(reader)
						return string(value), err
					}, fx.From(new(*bytes.Buffer))),
					fx.ParamTags(`name:"input"`), fx.ResultTags(`name:"output"`),
				),
			),
			fx.Populate(&result),
		)
		app.RequireStart().RequireStop()
		assert.Equal(t, "reader", result.Value)
	})

	t.Run("tagged result visible to outer hook", func(t *testing.T) {
		type result struct {
			fx.In
			Value *bytes.Buffer `name:"output"`
		}
		var populated result
		var hookValue *bytes.Buffer
		app := fxtest.New(t,
			fx.Provide(fx.Annotate(
				fx.Annotate(func() *bytes.Buffer { return bytes.NewBuffer(nil) }, fx.ResultTags(`name:"output"`)),
				fx.OnStart(func(value result) { hookValue = value.Value }),
			)),
			fx.Populate(&populated),
		)
		app.RequireStart().RequireStop()
		assert.Same(t, populated.Value, hookValue)
	})
}

func TestAnnotateNestedHookParameterAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("ParamTags apply to business parameter", func(t *testing.T) {
		var value, started int
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() string { return "input" }, fx.ResultTags(`name:"input"`)),
				fx.Annotate(
					fx.Annotate(func(value string) int { return len(value) },
						fx.OnStart(func(value int) { started = value }),
					),
					fx.ParamTags(`name:"input"`),
				),
			),
			fx.Populate(&value),
		)
		app.RequireStart().RequireStop()
		assert.Equal(t, 5, value)
		assert.Equal(t, value, started)
	})

	t.Run("From applies to business parameter", func(t *testing.T) {
		var value, started string
		app := fxtest.New(t,
			fx.Supply(bytes.NewBufferString("reader")),
			fx.Provide(fx.Annotate(
				fx.Annotate(func(reader io.Reader) (string, error) {
					value, err := io.ReadAll(reader)
					return string(value), err
				}, fx.OnStart(func(value string) { started = value })),
				fx.From(new(*bytes.Buffer)),
			)),
			fx.Populate(&value),
		)
		app.RequireStart().RequireStop()
		assert.Equal(t, "reader", value)
		assert.Equal(t, value, started)
	})

	t.Run("explicit Lifecycle retains its parameter position", func(t *testing.T) {
		var value string
		var events []string
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() int { return 42 }, fx.ResultTags(`name:"input"`)),
				fx.Annotate(
					fx.Annotate(func(lc fx.Lifecycle, input int) string {
						lc.Append(fx.Hook{OnStart: func(context.Context) error {
							events = append(events, "explicit lifecycle")
							return nil
						}})
						return fmt.Sprint(input)
					}, fx.OnStart(func() { events = append(events, "inner hook") })),
					fx.ParamTags("", `name:"input"`),
				),
			),
			fx.Populate(&value),
		)
		app.RequireStart().RequireStop()
		assert.Equal(t, "42", value)
		assert.Equal(t, []string{"explicit lifecycle", "inner hook"}, events)
	})
}

func TestAnnotateNestedMultipleResults(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBufferString("service")
	var calls int
	var result struct {
		fx.In
		Writer io.Writer `name:"writer"`
		Count  int       `name:"count"`
	}
	app := fxtest.New(t,
		fx.Provide(fx.Annotate(
			fx.Annotate(func() (*bytes.Buffer, int, error) {
				calls++
				return buffer, 42, nil
			}, fx.As(fx.Self(), fx.Self()), fx.ResultTags("", `name:"count"`)),
			fx.As(new(io.Writer)), fx.ResultTags(`name:"writer"`),
		)),
		fx.Populate(&result),
	)
	app.RequireStart().RequireStop()
	assert.Equal(t, 1, calls)
	assert.Same(t, buffer, result.Writer)
	assert.Equal(t, 42, result.Count)
}

func TestAnnotateNestedAsProjections(t *testing.T) {
	t.Parallel()

	value := bytes.NewBufferString("service")
	var calls int
	var result struct {
		fx.In
		Reader io.Reader `name:"reader"`
		Writer io.Writer `name:"writer"`
	}
	app := fxtest.New(t,
		fx.Provide(fx.Annotate(
			fx.Annotate(func() *bytes.Buffer {
				calls++
				return value
			}, fx.As(new(io.Reader)), fx.As(new(io.Writer))),
			fx.ResultTags(`name:"reader"`, `name:"writer"`),
		)),
		fx.Populate(&result),
	)
	app.RequireStart().RequireStop()
	assert.Equal(t, 1, calls)
	assert.Same(t, value, result.Reader)
	assert.Same(t, value, result.Writer)
}

func TestAnnotateNestedVariadic(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		group  bool
		values []int
		want   int
	}{
		{name: "no arguments"},
		{name: "empty group", group: true},
		{name: "value group", group: true, values: []int{3, 7}, want: 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var result, started, stopped int
			inner := fx.Annotate(func(values ...int) int {
				var sum int
				for _, value := range values {
					sum += value
				}
				return sum
			}, fx.OnStart(func(value int) { started = value }))
			annotations := []fx.Annotation{fx.OnStop(func(value int) { stopped = value })}
			if tt.group {
				annotations = append(annotations, fx.ParamTags(`group:"values"`))
			}
			options := []fx.Option{fx.Provide(fx.Annotate(inner, annotations...)), fx.Populate(&result)}
			for _, value := range tt.values {
				options = append(options, fx.Provide(fx.Annotate(func() int { return value }, fx.ResultTags(`group:"values"`))))
			}
			app := fxtest.New(t, options...)
			app.RequireStart().RequireStop()
			assert.Equal(t, tt.want, result)
			assert.Equal(t, tt.want, started)
			assert.Equal(t, tt.want, stopped)
		})
	}
}

func TestAnnotateNestedPopulate(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		annotations []fx.Annotation
		want        string
	}{
		{name: "inherit inner tag", want: "inner"},
		{name: "outer tag overrides inner", annotations: []fx.Annotation{fx.ParamTags(`name:"outer"`)}, want: "outer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			app := fxtest.New(t,
				fx.Provide(
					fx.Annotate(func() string { return "inner" }, fx.ResultTags(`name:"inner"`)),
					fx.Annotate(func() string { return "outer" }, fx.ResultTags(`name:"outer"`)),
				),
				fx.Populate(fx.Annotate(
					fx.Annotate(&result, fx.ParamTags(`name:"inner"`)),
					tt.annotations...,
				)),
			)
			app.RequireStart().RequireStop()
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAnnotateNestedOriginalStructRestrictions(t *testing.T) {
	t.Parallel()

	type input struct {
		fx.In
		Value string
	}
	type output struct {
		fx.Out
		Value *bytes.Buffer
	}
	for _, tt := range []struct {
		name       string
		target     any
		annotation fx.Annotation
		want       string
	}{
		{"In with ParamTags", func(input) int { return 1 }, fx.ParamTags(`name:"input"`), "fx.In structs cannot be annotated with fx.ParamTags or fx.From"},
		{"In with From", func(input) int { return 1 }, fx.From(new(input)), "fx.In structs cannot be annotated with fx.ParamTags or fx.From"},
		{"Out with ResultTags", func() output { return output{} }, fx.ResultTags(`name:"output"`), "fx.Out structs cannot be annotated with fx.ResultTags or fx.As"},
		{"Out with As", func() output { return output{} }, fx.As(new(io.Writer)), "fx.Out structs cannot be annotated with fx.ResultTags or fx.As"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := fx.New(fx.NopLogger, fx.Provide(fx.Annotate(
				fx.Annotate(tt.target, fx.OnStart(func() {})),
				tt.annotation,
			)))
			require.ErrorContains(t, app.Err(), tt.want)
		})
	}
}

func TestAnnotateNestedFromConcreteParameter(t *testing.T) {
	t.Parallel()

	var err error
	require.NotPanics(t, func() {
		app := fx.New(fx.NopLogger, fx.Provide(fx.Annotate(
			fx.Annotate(func(io.Reader) int { return 1 }, fx.From(new(*bytes.Buffer))),
			fx.From(new(*bytes.Buffer)),
		)))
		err = app.Err()
	})
	require.Error(t, err)
}

func TestAnnotateNestedValues(t *testing.T) {
	t.Parallel()

	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("Replace=%v", replace), func(t *testing.T) {
			value := bytes.NewBufferString("replacement")
			var result, started, stopped *bytes.Buffer
			annotated := fx.Annotate(
				fx.Annotate(value, fx.OnStart(func(value *bytes.Buffer) { started = value })),
				fx.OnStop(func(value *bytes.Buffer) { stopped = value }),
			)
			var options []fx.Option
			if replace {
				options = []fx.Option{
					fx.Supply(bytes.NewBufferString("original")),
					fx.Replace(annotated),
				}
			} else {
				options = []fx.Option{fx.Supply(annotated)}
			}
			options = append(options, fx.Populate(&result))
			app := fxtest.New(t, options...)
			app.RequireStart().RequireStop()
			assert.Same(t, value, result)
			assert.Same(t, value, started)
			assert.Same(t, value, stopped)
		})
	}
}

func TestAnnotateNestedDecorator(t *testing.T) {
	t.Parallel()

	original := bytes.NewBufferString("original")
	var result, started, stopped *bytes.Buffer
	var calls int
	app := fxtest.New(t,
		fx.Supply(original),
		fx.Decorate(fx.Annotate(
			fx.Annotate(func(value *bytes.Buffer) *bytes.Buffer {
				calls++
				return bytes.NewBufferString(value.String() + " decorated")
			}, fx.OnStart(func(value *bytes.Buffer) { started = value })),
			fx.OnStop(func(value *bytes.Buffer) { stopped = value }),
		)),
		fx.Populate(&result),
	)
	app.RequireStart().RequireStop()
	assert.Equal(t, 1, calls)
	assert.Equal(t, "original decorated", result.String())
	assert.NotSame(t, original, result)
	assert.Same(t, result, started)
	assert.Same(t, result, stopped)
}

func TestAnnotateNestedEmptyResultTags(t *testing.T) {
	t.Parallel()

	failure := errors.New("constructor failed")
	for _, constructorError := range []error{nil, failure} {
		t.Run(fmt.Sprintf("error=%v", constructorError), func(t *testing.T) {
			value := bytes.NewBufferString("service")
			var calls int
			var result *bytes.Buffer
			app := fx.New(fx.NopLogger,
				fx.Provide(fx.Annotate(
					fx.Annotate(func() (*bytes.Buffer, error) {
						calls++
						return value, constructorError
					}, fx.ResultTags()),
					fx.ResultTags(),
				)),
				fx.Populate(&result),
			)
			if constructorError != nil {
				require.ErrorIs(t, app.Err(), constructorError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, app.Err())
				require.NoError(t, app.Start(context.Background()))
				assert.Same(t, value, result)
			}
			require.NoError(t, app.Stop(context.Background()))
			assert.Equal(t, 1, calls)
		})
	}
}

func TestAnnotateNestedReuse(t *testing.T) {
	t.Parallel()

	var calls int
	outerStarts, outerStops := make(map[string]int), make(map[string]int)
	starts, stops := make(map[*bytes.Buffer]int), make(map[*bytes.Buffer]int)
	inner := fx.Annotate(func() *bytes.Buffer {
		calls++
		return bytes.NewBuffer(nil)
	}, fx.OnStart(func(value *bytes.Buffer) { starts[value]++ }), fx.OnStop(func(value *bytes.Buffer) { stops[value]++ }))
	first := fx.Annotate(inner, fx.ResultTags(`name:"first"`),
		fx.OnStart(func() { outerStarts["first"]++ }), fx.OnStop(func() { outerStops["first"]++ }))
	second := fx.Annotate(inner, fx.ResultTags(`name:"second"`),
		fx.OnStart(func() { outerStarts["second"]++ }), fx.OnStop(func() { outerStops["second"]++ }))
	for i := 0; i < 2; i++ {
		var result struct {
			fx.In
			Original *bytes.Buffer
			First    *bytes.Buffer `name:"first"`
			Second   *bytes.Buffer `name:"second"`
		}
		app := fxtest.New(t, fx.Provide(inner, first, second), fx.Populate(&result))
		assert.NotSame(t, result.Original, result.First)
		assert.NotSame(t, result.First, result.Second)
		app.RequireStart().RequireStop()
		assert.Equal(t, 3*(i+1), calls)
		assert.Equal(t, map[string]int{"first": i + 1, "second": i + 1}, outerStarts)
		assert.Equal(t, map[string]int{"first": i + 1, "second": i + 1}, outerStops)
	}
	require.Len(t, starts, 6)
	require.Len(t, stops, 6)
	for value, count := range starts {
		assert.Equal(t, 1, count)
		assert.Equal(t, 1, stops[value])
	}
}

func TestAnnotateNestedParallelReuse(t *testing.T) {
	t.Parallel()

	type service struct {
		starts atomic.Int32
		stops  atomic.Int32
	}
	var calls atomic.Int32
	target := fx.Annotate(
		fx.Annotate(func() *service {
			calls.Add(1)
			return &service{}
		}, fx.OnStart(func(value *service) { value.starts.Add(1) })),
		fx.OnStop(func(value *service) { value.stops.Add(1) }),
	)
	t.Cleanup(func() { assert.Equal(t, int32(4), calls.Load()) })
	for i := 0; i < 4; i++ {
		t.Run(fmt.Sprintf("app%d", i), func(t *testing.T) {
			t.Parallel()
			var value *service
			app := fxtest.New(t, fx.Provide(target), fx.Populate(&value))
			app.RequireStart().RequireStop()
			assert.Equal(t, int32(1), value.starts.Load())
			assert.Equal(t, int32(1), value.stops.Load())
		})
	}
}

func TestAnnotateNestedConstructorError(t *testing.T) {
	t.Parallel()

	failure := errors.New("constructor failed")
	var calls, hooks int
	target := fx.Annotate(
		fx.Annotate(func() (*bytes.Buffer, error) {
			calls++
			return nil, failure
		}, fx.OnStart(func() { hooks++ }), fx.OnStop(func() { hooks++ })),
		fx.OnStart(func() { hooks++ }), fx.OnStop(func() { hooks++ }),
	)
	app := fx.New(fx.NopLogger, fx.Provide(target), fx.Invoke(func(*bytes.Buffer) {}))
	require.ErrorIs(t, app.Err(), failure)
	require.ErrorIs(t, app.Start(context.Background()), failure)
	require.NoError(t, app.Stop(context.Background()))
	assert.Equal(t, 1, calls)
	assert.Zero(t, hooks)
}

func TestAnnotateNestedStartError(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		failInner bool
		want      []string
	}{
		{"inner", true, []string{"dependency start", "inner start", "dependency stop"}},
		{"outer", false, []string{"dependency start", "inner start", "outer start", "inner stop", "dependency stop"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			type dependency struct{}
			failure := errors.New("start failed")
			var events []string
			app := fx.New(fx.NopLogger,
				fx.Provide(func(lc fx.Lifecycle) *dependency {
					lc.Append(fx.Hook{
						OnStart: func(context.Context) error {
							events = append(events, "dependency start")
							return nil
						},
						OnStop: func(context.Context) error {
							events = append(events, "dependency stop")
							return nil
						},
					})
					return &dependency{}
				}),
				fx.Provide(fx.Annotate(
					fx.Annotate(func(*dependency) *bytes.Buffer { return bytes.NewBuffer(nil) },
						fx.OnStart(func() error {
							events = append(events, "inner start")
							if tt.failInner {
								return failure
							}
							return nil
						}),
						fx.OnStop(func() { events = append(events, "inner stop") }),
					),
					fx.OnStart(func() error {
						events = append(events, "outer start")
						return failure
					}),
					fx.OnStop(func() { events = append(events, "outer stop") }),
				)),
				fx.Invoke(func(*bytes.Buffer) {}),
			)
			require.NoError(t, app.Err())
			require.ErrorIs(t, app.Start(context.Background()), failure)
			assert.Equal(t, tt.want, events)
			require.NoError(t, app.Stop(context.Background()))
			assert.Equal(t, tt.want, events, "rolled-back hooks must not stop twice")
		})
	}
}

func TestAnnotateNestedStopErrors(t *testing.T) {
	t.Parallel()

	innerFailure, outerFailure := errors.New("inner stop failed"), errors.New("outer stop failed")
	var events []string
	app := fxtest.New(t,
		fx.Provide(fx.Annotate(
			fx.Annotate(func() *bytes.Buffer { return bytes.NewBuffer(nil) },
				fx.OnStart(func() { events = append(events, "inner start") }),
				fx.OnStop(func() error {
					events = append(events, "inner stop")
					return innerFailure
				}),
			),
			fx.OnStart(func() { events = append(events, "outer start") }),
			fx.OnStop(func() error {
				events = append(events, "outer stop")
				return outerFailure
			}),
		)),
		fx.Invoke(func(*bytes.Buffer) {}),
	)
	app.RequireStart()
	err := app.Stop(context.Background())
	require.ErrorIs(t, err, innerFailure)
	require.ErrorIs(t, err, outerFailure)
	assert.Equal(t, []string{"inner start", "outer start", "outer stop", "inner stop"}, events)
}

func TestAnnotateNestedInvalidAnnotations(t *testing.T) {
	t.Parallel()

	constructor := func() int { return 1 }
	start := fx.OnStart(func() {})
	stop := fx.OnStop(func() {})
	for _, tt := range []struct {
		name   string
		target any
		want   string
	}{
		{"invalid inner hook", fx.Annotate(fx.Annotate(constructor, fx.OnStart(42)), stop), "must provide function"},
		{"invalid outer hook", fx.Annotate(fx.Annotate(constructor, start), fx.OnStop(42)), "must provide function"},
		{"invalid inner As", fx.Annotate(fx.Annotate(constructor, fx.As(new(io.Writer))), start), "int does not implement io.Writer"},
		{"invalid outer As", fx.Annotate(fx.Annotate(constructor, start), fx.As(new(io.Writer))), "int does not implement io.Writer"},
		{"duplicate start in one layer", fx.Annotate(constructor, start, start), `cannot apply more than one "OnStart" hook annotation`},
		{"duplicate stop in one layer", fx.Annotate(constructor, stop, stop), `cannot apply more than one "OnStop" hook annotation`},
		{"duplicate start in inner layer", fx.Annotate(fx.Annotate(constructor, start, start), stop), `cannot apply more than one "OnStart" hook annotation`},
		{"duplicate stop in outer layer", fx.Annotate(fx.Annotate(constructor, start), stop, stop), `cannot apply more than one "OnStop" hook annotation`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := fx.New(fx.NopLogger, fx.Provide(tt.target))
			require.ErrorContains(t, app.Err(), tt.want)
		})
	}
}
