// Copyright (c) 2023 Uber Technologies, Inc.
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
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/fx/internal/fxclock"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/goleak"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func NewForTest(tb testing.TB, opts ...Option) *App {
	testOpts := []Option{
		// Provide both: Logger and WithLogger so that if the test
		// WithLogger fails, we don't pollute stderr.
		Logger(fxtest.NewTestPrinter(tb)),
		fxtest.WithTestLogger(tb),
	}
	opts = append(testOpts, opts...)

	return New(opts...)
}

func NewSpied(opts ...Option) (*App, *fxlog.Spy) {
	spy := new(fxlog.Spy)
	opts = append([]Option{
		WithLogger(func() fxevent.Logger { return spy }),
	}, opts...)
	return New(opts...), spy
}

func validateTestApp(tb testing.TB, opts ...Option) error {
	testOpts := []Option{
		// Provide both: Logger and WithLogger so that if the test
		// WithLogger fails, we don't pollute stderr.
		Logger(fxtest.NewTestPrinter(tb)),
		fxtest.WithTestLogger(tb),
	}
	opts = append(testOpts, opts...)

	return ValidateApp(opts...)
}

func TestNewApp(t *testing.T) {
	t.Parallel()

	t.Run("ProvidesLifecycleAndShutdowner", func(t *testing.T) {
		t.Parallel()

		var (
			l Lifecycle
			s Shutdowner
		)
		fxtest.New(
			t,
			Populate(&l, &s),
		)
		assert.NotNil(t, l)
		assert.NotNil(t, s)
	})

	t.Run("OptionsHappensBeforeProvides", func(t *testing.T) {
		t.Parallel()

		// Should be grouping all provides and pushing them into the container
		// after applying other options. This prevents the app configuration
		// (e.g., logging) from changing halfway through our provides.

		spy := new(fxlog.Spy)
		app := fxtest.New(t, Provide(func() struct{} { return struct{}{} }),
			WithLogger(func() fxevent.Logger { return spy }))
		defer app.RequireStart().RequireStop()
		require.Equal(t,
			[]string{"Provided", "Provided", "Provided", "Provided", "LoggerInitialized", "Started"},
			spy.EventTypes())

		// Fx types get provided first to increase chance of
		// successful custom logger build.
		assert.Contains(t, spy.Events()[0].(*fxevent.Provided).OutputTypeNames, "fx.Lifecycle")
		assert.Contains(t, spy.Events()[1].(*fxevent.Provided).OutputTypeNames, "fx.Shutdowner")
		assert.Contains(t, spy.Events()[2].(*fxevent.Provided).OutputTypeNames, "fx.DotGraph")
		// Our type should be index 3.
		assert.Contains(t, spy.Events()[3].(*fxevent.Provided).OutputTypeNames, "struct {}")
	})

	t.Run("CircularGraphReturnsError", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}
		app := NewForTest(t,
			Provide(func(A) B { return B{} }),
			Provide(func(B) A { return A{} }),
			Invoke(func(B) {}),
		)
		err := app.Err()
		require.Error(t, err, "fx.New should return an error")

		errMsg := err.Error()
		assert.Contains(t, errMsg, "cycle detected in dependency graph")
		assert.Contains(t, errMsg, "depends on func(fx_test.B) fx_test.A")
		assert.Contains(t, errMsg, "depends on func(fx_test.A) fx_test.B")
	})

	t.Run("ProvidesDotGraph", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}
		type C struct{}
		var g DotGraph
		app := fxtest.New(t,
			Provide(func() A { return A{} }),
			Provide(func(A) B { return B{} }),
			Provide(func(A, B) C { return C{} }),
			Populate(&g),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
		assert.Contains(t, g, `"fx.DotGraph" [label=<fx.DotGraph>];`)
	})

	t.Run("ProvidesWithAnnotate", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		type B struct {
			In

			Foo  A   `name:"foo"`
			Bar  A   `name:"bar"`
			Foos []A `group:"foo"`
		}

		app := fxtest.New(t,
			Provide(
				Annotated{
					Target: func() A { return A{} },
					Name:   "foo",
				},
				Annotated{
					Target: func() A { return A{} },
					Name:   "bar",
				},
				Annotated{
					Target: func() A { return A{} },
					Group:  "foo",
				},
			),
			Invoke(
				func(b B) {
					assert.NotNil(t, b.Foo)
					assert.NotNil(t, b.Bar)
					assert.Len(t, b.Foos, 1)
				},
			),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("ProvidesWithAnnotateFlattened", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			Provide(Annotated{
				Target: func() []int { return []int{1} },
				Group:  "foo,flatten",
			}),
			Invoke(
				func(b struct {
					In
					Foos []int `group:"foo"`
				},
				) {
					assert.Len(t, b.Foos, 1)
				},
			),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("ProvidesWithEmptyAnnotate", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		type B struct {
			In

			Foo A
		}

		app := fxtest.New(t,
			Provide(
				Annotated{
					Target: func() A { return A{} },
				},
			),
			Invoke(
				func(b B) {
					assert.NotNil(t, b.Foo)
				},
			),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("CannotNameAndGroup", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		app := NewForTest(t,
			Provide(
				Annotated{
					Target: func() A { return A{} },
					Name:   "foo",
					Group:  "bar",
				},
			),
		)

		err := app.Err()
		require.Error(t, err)

		// fx.Annotated may specify only one of Name or Group: received fx.Annotated{Name: "foo", Group: "bar", Target: go.uber.org/fx_test.TestAnnotatedWithGroupAndName.func1()} from:
		// go.uber.org/fx_test.TestAnnotatedWithGroupAndName
		//         /.../fx/annotated_test.go:164
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Annotated may specify only one of Name or Group:")
		assert.Contains(t, err.Error(), `received fx.Annotated{Name: "foo", Group: "bar", Target: go.uber.org/fx_test.TestNewApp`)
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestNewApp")
		assert.Contains(t, err.Error(), "/app_test.go")
	})

	t.Run("ErrorProvidingAnnotated", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t, Provide(Annotated{
			Target: 42, // not a constructor
			Name:   "foo",
		}))

		err := app.Err()
		require.Error(t, err)

		// Example:
		// fx.Provide(fx.Annotated{...}) from:
		//     go.uber.org/fx_test.TestNewApp.func8
		//         /.../fx/app_test.go:206
		//     testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		//     Failed: must provide constructor function, got 42 (type int)
		assert.Contains(t, err.Error(), `fx.Provide(fx.Annotated{Name: "foo", Target: 42}) from:`)
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestNewApp")
		assert.Contains(t, err.Error(), "/app_test.go")
		assert.Contains(t, err.Error(), "Failed: must provide constructor function")
	})

	t.Run("ErrorProvidingAnnotate", func(t *testing.T) {
		t.Parallel()

		type t1 struct{}
		newT1 := func() t1 { return t1{} }

		// Provide twice.
		app := NewForTest(t, Provide(
			Annotate(newT1, ResultTags(`name:"foo"`)),
			Annotate(newT1, ResultTags(`name:"foo"`)),
		))

		err := app.Err()
		require.Error(t, err)

		// Example:
		// fx.Provide(fx.Annotate(go.uber.org/fx_test.TestNewApp.func10.1(), fx.ResultTags(["name:\"foo\""])) from:
		//     go.uber.org/fx_test.TestNewApp.func10
		//         /.../fx/app_test.go:305
		//     testing.tRunner
		//         /.../src/testing/testing.go:1259
		//     Failed: cannot provide function "reflect".makeFuncStub (/.../reflect/asm_amd64.s:30):
		//     cannot provide fx_test.t1[name="foo"] from [0].Field0:
		//     already provided by "reflect".makeFuncStub (/.../reflect/asm_amd64.s:30)
		assert.Contains(t, err.Error(), `fx.Provide(fx.Annotate(`)
		assert.Contains(t, err.Error(), `fx.ResultTags([["name:\"foo\""]])`)
		assert.Contains(t, err.Error(), "already provided")
	})

	t.Run("ErrorProviding", func(t *testing.T) {
		t.Parallel()

		err := NewForTest(t, Provide(42)).Err()
		require.Error(t, err)

		// Example:
		// fx.Provide(..) from:
		//     go.uber.org/fx_test.TestNewApp.func8
		//         /.../fx/app_test.go:206
		//     testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		//     Failed: must provide constructor function, got 42 (type int)
		assert.Contains(t, err.Error(), "fx.Provide(42) from:")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestNewApp")
		assert.Contains(t, err.Error(), "/app_test.go")
		assert.Contains(t, err.Error(), "Failed: must provide constructor function")
	})

	t.Run("Decorates", func(t *testing.T) {
		t.Parallel()
		spy := new(fxlog.Spy)

		type A struct{ value int }
		app := fxtest.New(t,
			Provide(func() A { return A{value: 0} }),
			Decorate(func(a A) A { return A{value: a.value + 1} }),
			Invoke(func(a A) { assert.Equal(t, a.value, 1) }),
			WithLogger(func() fxevent.Logger { return spy }))
		defer app.RequireStart().RequireStop()

		require.Equal(t,
			[]string{"Provided", "Provided", "Provided", "Provided", "Decorated", "LoggerInitialized", "Invoking", "Run", "Run", "Invoked", "Started"},
			spy.EventTypes())
	})

	t.Run("DecoratesFromManyModules", func(t *testing.T) {
		t.Parallel()
		spy := new(fxlog.Spy)

		type A struct{ value int }
		m := Module("decorator",
			Decorate(func(a A) A { return A{value: a.value + 1} }),
		)
		app := fxtest.New(t,
			m,
			Provide(func() A { return A{value: 0} }),
			Decorate(func(a A) A { return A{value: a.value + 1} }),
			WithLogger(func() fxevent.Logger { return spy }),
		)
		defer app.RequireStart().RequireStop()

		require.Equal(t,
			[]string{"Provided", "Provided", "Provided", "Provided", "Decorated", "Decorated", "LoggerInitialized", "Started"},
			spy.EventTypes())
	})
}

// TestPrivate tests Private when used with both fx.Provide and fx.Supply.
func TestPrivate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc string

		// provide is either a Supply or Provide wrapper around the given int
		// that allows us to generalize these test cases for both APIs.
		provide func(int, bool) Option
	}{
		{
			desc: "Provide",
			provide: func(i int, private bool) Option {
				opts := []any{func() int { return i }}
				if private {
					opts = append(opts, Private)
				}
				return Provide(opts...)
			},
		},
		{
			desc: "Supply",
			provide: func(i int, private bool) Option {
				opts := []any{i}
				if private {
					opts = append(opts, Private)
				}
				return Supply(opts...)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			t.Run("CanUsePrivateFromParentModule", func(t *testing.T) {
				t.Parallel()

				var invoked bool
				app := fxtest.New(t,
					Module("SubModule", Invoke(func(a int, b string) {
						assert.Equal(t, 0, a)
						invoked = true
					})),
					Provide(func() string { return "" }),
					tt.provide(0, true /* private */),
				)
				app.RequireStart().RequireStop()
				assert.True(t, invoked)
			})

			t.Run("CannotUsePrivateFromSubModule", func(t *testing.T) {
				t.Parallel()

				app := NewForTest(t,
					Module("SubModule", tt.provide(0, true /* private */)),
					Invoke(func(a int) {}),
				)
				err := app.Err()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing dependencies for function")
				assert.Contains(t, err.Error(), "missing type: int")
			})

			t.Run("MultipleModulesSameType", func(t *testing.T) {
				t.Parallel()

				var invoked int
				app := fxtest.New(t,
					Module("SubModuleA",
						tt.provide(1, true /* private */),
						Invoke(func(s int) {
							assert.Equal(t, 1, s)
							invoked++
						}),
					),
					Module("SubModuleB",
						tt.provide(2, true /* private */),
						Invoke(func(s int) {
							assert.Equal(t, 2, s)
							invoked++
						}),
					),
					tt.provide(3, false /* private */),
					Invoke(func(s int) {
						assert.Equal(t, 3, s)
						invoked++
					}),
				)
				app.RequireStart().RequireStop()
				assert.Equal(t, 3, invoked)
			})
		})
	}
}

func TestPrivateProvideWithDecorators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc string

		// provide is either a Supply or Provide wrapper around the given int
		// that allows us to generalize these test cases for both APIs.
		provide func(int) Option
		private bool
	}{
		{
			desc: "Private/Provide",
			provide: func(i int) Option {
				return Provide(
					func() int { return i },
					Private,
				)
			},
			private: true,
		},
		{
			desc: "Private/Supply",
			provide: func(i int) Option {
				return Supply(i, Private)
			},
			private: true,
		},
		{
			desc: "Public/Provide",
			provide: func(i int) Option {
				return Provide(func() int { return i })
			},
			private: false,
		},
		{
			desc: "Public/Supply",
			provide: func(i int) Option {
				return Supply(i)
			},
			private: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			t.Run("DecoratedTypeInSubModule", func(t *testing.T) {
				t.Parallel()

				var invoked bool
				app := NewForTest(t,
					Module("SubModule",
						tt.provide(0),
						Decorate(func(a int) int { return a + 2 }),
						Invoke(func(a int) { assert.Equal(t, 2, a) }),
					),
					Invoke(func(a int) {
						// Decoration is always "private",
						// so raw provided value is expected here
						// when the submodule provides it as public.
						assert.Equal(t, 0, a)
						invoked = true
					}),
				)
				err := app.Err()
				if tt.private {
					require.Error(t, err)
					assert.Contains(t, err.Error(), "missing dependencies for function")
					assert.Contains(t, err.Error(), "missing type: int")
				} else {
					require.NoError(t, err)
					assert.True(t, invoked)
				}
			})

			t.Run("DecoratedTypeInParentModule", func(t *testing.T) {
				t.Parallel()

				var invoked int
				app := fxtest.New(t,
					tt.provide(0),
					Decorate(func(a int) int { return a - 5 }),
					Invoke(func(a int) {
						assert.Equal(t, -5, a)
						invoked++
					}),
					Module("Child",
						Decorate(func(a int) int { return a + 10 }),
						Invoke(func(a int) {
							assert.Equal(t, 5, a)
							invoked++
						}),
					),
				)
				app.RequireStart().RequireStop()
				assert.Equal(t, 2, invoked)
			})

			t.Run("ParentDecorateChildType", func(t *testing.T) {
				t.Parallel()

				var invoked bool
				app := NewForTest(t,
					Module("Child", tt.provide(0)),
					Decorate(func(a int) int { return a + 5 }),
					Invoke(func(a int) {
						invoked = true
					}),
				)
				err := app.Err()
				if tt.private {
					require.Error(t, err)
					assert.Contains(t, err.Error(), "missing dependencies for function")
					assert.Contains(t, err.Error(), "missing type: int")
				} else {
					require.NoError(t, err)
					assert.True(t, invoked)
				}
			})
		})
	}
}

func TestWithLoggerErrorUseDefault(t *testing.T) {
	// This test cannot be run in paralllel with the others because
	// it hijacks stderr.

	// Temporarily hijack stderr and restore it after this test so
	// that we can assert its contents.
	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatalf("could not open a file for writing")
	}
	defer func(oldStderr *os.File) {
		assert.NoError(t, f.Close())
		os.Stderr = oldStderr
	}(os.Stderr)
	os.Stderr = f

	app := New(
		Supply(zap.NewNop()),
		WithLogger(&bytes.Buffer{}),
	)
	err = app.Err()
	require.Error(t, err)
	assert.Contains(t,
		err.Error(),
		"must provide constructor function, got  (type *bytes.Buffer)",
	)

	stderr, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	// Example output:
	// [Fx] SUPPLY  *zap.Logger
	// [Fx] ERROR   Failed to initialize custom logger: fx.WithLogger() from:
	// go.uber.org/fx_test.TestSetupLogger.func3
	//        /Users/abg/dev/fx/app_test.go:334
	// testing.tRunner
	//        /usr/local/Cellar/go/1.16.4/libexec/src/testing/testing.go:1193
	// Failed: must provide constructor function, got  (type *bytes.Buffer)

	out := string(stderr)
	assert.Contains(t, out, "[Fx] SUPPLY\t*zap.Logger\n")
	assert.Contains(t, out, "[Fx] ERROR\t\tFailed to initialize custom logger: fx.WithLogger")
	assert.Contains(t, out, "must provide constructor function, got  (type *bytes.Buffer)\n")
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	t.Run("initializing custom logger", func(t *testing.T) {
		t.Parallel()

		var spy fxlog.Spy
		app := fxtest.New(t,
			Supply(&spy),
			WithLogger(func(spy *fxlog.Spy) fxevent.Logger {
				return spy
			}),
		)

		assert.Equal(t, []string{
			"Provided", "Provided", "Provided", "Supplied", "Run", "LoggerInitialized",
		}, spy.EventTypes())

		spy.Reset()
		app.RequireStart().RequireStop()

		require.NoError(t, app.Err())

		assert.Equal(t, []string{"Started", "Stopped"}, spy.EventTypes())
	})

	t.Run("error in Provide shows logs", func(t *testing.T) {
		t.Parallel()

		var spy fxlog.Spy
		app := New(
			Supply(&spy),
			WithLogger(func(spy *fxlog.Spy) fxevent.Logger {
				return spy
			}),
			Provide(&bytes.Buffer{}), // not passing in a constructor.
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t,
			err.Error(),
			"must provide constructor function, got  (type *bytes.Buffer)",
		)

		assert.Equal(t, []string{"Provided", "Provided", "Provided", "Supplied", "Provided", "Run", "LoggerInitialized"}, spy.EventTypes())
	})

	t.Run("logger failed to build", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		app := New(
			Logger(log.New(&buff, "", 0)),
			WithLogger(func() (fxevent.Logger, error) {
				return nil, errors.New("great sadness")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "great sadness")

		out := buff.String()
		assert.Contains(t, out, "[Fx] ERROR\t\tFailed to initialize custom logger")
	})

	t.Run("logger dependency failed to build", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		app := New(
			Logger(log.New(&buff, "", 0)),
			Provide(func() (*zap.Logger, error) {
				return nil, errors.New("great sadness")
			}),
			WithLogger(func(log *zap.Logger) fxevent.Logger {
				t.Errorf("WithLogger must not be called")
				panic("must not be called")
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "great sadness")

		out := buff.String()
		assert.Contains(t, out, "[Fx] PROVIDE\t*zap.Logger")
		assert.Contains(t, out, "[Fx] ERROR\t\tFailed to initialize custom logger")
	})
}

func getInt() int { return 0 }

func decorateInt(i int) int { return i }

var moduleA = Module(
	"ModuleA",
	Provide(getInt),
	Decorate(decorateInt),
	Supply(int64(14)),
	Replace("foo"),
)

func getModuleB() Option {
	return Module(
		"ModuleB",
		moduleA,
	)
}

func TestModuleTrace(t *testing.T) {
	t.Parallel()

	moduleC := Module(
		"ModuleC",
		getModuleB(),
	)

	app, spy := NewSpied(moduleC)
	require.NoError(t, app.Err())

	wantTrace, err := regexp.Compile(
		// Provide/decorate itself, initialized via init.
		"^go.uber.org/fx_test.init \\(.*fx/app_test.go:.*\\)\n" +
			// ModuleA initialized via init.
			"go.uber.org/fx_test.init \\(.*fx/app_test.go:.*\\) \\(ModuleA\\)\n" +
			// ModuleB from getModuleB.
			"go.uber.org/fx_test.getModuleB \\(.*fx/app_test.go:.*\\) \\(ModuleB\\)\n" +
			// ModuleC above.
			"go.uber.org/fx_test.TestModuleTrace \\(.*fx/app_test.go:.*\\) \\(ModuleC\\)\n" +
			// Top-level app & corresponding module created by NewSpied.
			"go.uber.org/fx_test.NewSpied \\(.*fx/app_test.go:.*\\)$",
	)
	require.NoError(t, err, "test regexp compilation error")

	for _, tt := range []struct {
		desc     string
		getTrace func(t *testing.T) []string
	}{
		{
			desc: "Provide",
			getTrace: func(t *testing.T) []string {
				t.Helper()
				var event *fxevent.Provided
				for _, e := range spy.Events().SelectByTypeName("Provided") {
					pe, ok := e.(*fxevent.Provided)
					if !ok {
						continue
					}

					if strings.HasSuffix(pe.ConstructorName, "getInt()") {
						event = pe
						break
					}
				}
				require.NotNil(t, event, "could not find provide event for getInt()")
				return event.ModuleTrace
			},
		},
		{
			desc: "Decorate",
			getTrace: func(t *testing.T) []string {
				t.Helper()
				events := spy.Events().SelectByTypeName("Decorated")
				require.Len(t, events, 1)
				event, ok := events[0].(*fxevent.Decorated)
				require.True(t, ok)
				return event.ModuleTrace
			},
		},
		{
			desc: "Supply",
			getTrace: func(t *testing.T) []string {
				t.Helper()
				events := spy.Events().SelectByTypeName("Supplied")
				require.Len(t, events, 1)
				event, ok := events[0].(*fxevent.Supplied)
				require.True(t, ok)
				return event.ModuleTrace
			},
		},
		{
			desc: "Replaced",
			getTrace: func(t *testing.T) []string {
				t.Helper()
				events := spy.Events().SelectByTypeName("Replaced")
				require.Len(t, events, 1)
				event, ok := events[0].(*fxevent.Replaced)
				require.True(t, ok)
				return event.ModuleTrace
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			gotTrace := strings.Join(tt.getTrace(t), "\n")
			assert.Regexp(t, wantTrace, gotTrace)
		})
	}
}

func TestRunEventEmission(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc          string
		giveOpts      []Option
		wantRunEvents []fxevent.Run
		wantErr       string
	}{
		{
			desc: "Simple Provide And Decorate",
			giveOpts: []Option{
				Provide(func() int { return 5 }),
				Decorate(func(int) int { return 6 }),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func1()",
					Kind: "provide",
				},
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func2()",
					Kind: "decorate",
				},
			},
		},
		{
			desc: "Supply and Decorator Error",
			giveOpts: []Option{
				Supply(5),
				Decorate(func(int) (int, error) {
					return 0, errors.New("humongous despair")
				}),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "stub(int)",
					Kind: "supply",
				},
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func4()",
					Kind: "decorate",
				},
			},
			wantErr: "humongous despair",
		},
		{
			desc: "Replace",
			giveOpts: []Option{
				Provide(func() int { return 5 }),
				Replace(6),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "stub(int)",
					Kind: "replace",
				},
			},
		},
		{
			desc: "Provide Error",
			giveOpts: []Option{
				Provide(func() (int, error) {
					return 0, errors.New("terrible sadness")
				}),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func8()",
					Kind: "provide",
				},
			},
			wantErr: "terrible sadness",
		},
		{
			desc: "Provide Panic",
			giveOpts: []Option{
				Provide(func() int {
					panic("bad provide")
				}),
				RecoverFromPanics(),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func10()",
					Kind: "provide",
				},
			},
			wantErr: `panic: "bad provide"`,
		},
		{
			desc: "Decorate Panic",
			giveOpts: []Option{
				Supply(5),
				Decorate(func(int) int {
					panic("bad decorate")
				}),
				RecoverFromPanics(),
				Invoke(func(int) {}),
			},
			wantRunEvents: []fxevent.Run{
				{
					Name: "stub(int)",
					Kind: "supply",
				},
				{
					Name: "go.uber.org/fx_test.TestRunEventEmission.func12()",
					Kind: "decorate",
				},
			},
			wantErr: `panic: "bad decorate"`,
		},
	} {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			app, spy := NewSpied(tt.giveOpts...)
			if tt.wantErr != "" {
				assert.ErrorContains(t, app.Err(), tt.wantErr)
			} else {
				assert.NoError(t, app.Err())
			}

			gotEvents := spy.Events().SelectByTypeName("Run")
			require.Len(t, gotEvents, len(tt.wantRunEvents))
			for i, event := range gotEvents {
				rEvent, ok := event.(*fxevent.Run)
				require.True(t, ok)

				assert.Equal(t, tt.wantRunEvents[i].Name, rEvent.Name)
				assert.Equal(t, tt.wantRunEvents[i].Kind, rEvent.Kind)
				if tt.wantErr != "" && i == len(gotEvents)-1 {
					assert.ErrorContains(t, rEvent.Err, tt.wantErr)
				} else {
					assert.NoError(t, rEvent.Err)
				}
			}
		})
	}
}

type customError struct {
	err error
}

func (e *customError) Error() string {
	return fmt.Sprintf("custom error: %v", e.err)
}

func (e *customError) Unwrap() error {
	return e.err
}

type errHandlerFunc func(error)

func (f errHandlerFunc) HandleError(err error) { f(err) }

func TestInvokes(t *testing.T) {
	t.Parallel()

	t.Run("Success event", func(t *testing.T) {
		t.Parallel()

		app, spy := NewSpied(
			Invoke(func() {}),
		)
		require.NoError(t, app.Err())

		invoked := spy.Events().SelectByTypeName("Invoked")
		require.Len(t, invoked, 1)
		assert.NoError(t, invoked[0].(*fxevent.Invoked).Err)
	})

	t.Run("Failure event", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("great sadness")
		app, spy := NewSpied(
			Invoke(func() error {
				return wantErr
			}),
		)
		require.Error(t, app.Err())

		invoked := spy.Events().SelectByTypeName("Invoked")
		require.Len(t, invoked, 1)
		require.Error(t, invoked[0].(*fxevent.Invoked).Err)
		require.ErrorIs(t, invoked[0].(*fxevent.Invoked).Err, wantErr)
	})

	t.Run("ErrorsAreNotOverriden", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		app := NewForTest(t,
			Provide(func() B { return B{} }), // B inserted into the graph
			Invoke(func(A) {}),               // failed A invoke
			Invoke(func(B) {}),               // successful B invoke
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing type: fx_test.A")
	})

	t.Run("ErrorHooksAreCalled", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		count := 0
		h := errHandlerFunc(func(err error) {
			count++
		})
		NewForTest(t,
			Invoke(func(A) {}),
			ErrorHook(h),
		)
		assert.Equal(t, 1, count)
	})

	t.Run("ErrorsAreWrapped", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("err")

		app := NewForTest(t,
			Invoke(func() error {
				return &customError{err: wantErr}
			}),
		)

		err := app.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, wantErr)
		var ce *customError
		assert.ErrorAs(t, err, &ce)
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("NilErrorOption", func(t *testing.T) {
		t.Parallel()

		var invoked bool

		app := NewForTest(t,
			Error(nil),
			Invoke(func() { invoked = true }),
		)
		err := app.Err()
		require.NoError(t, err)
		assert.True(t, invoked)
	})

	t.Run("SingleErrorOption", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("module failure")
		app := NewForTest(t,
			Error(wantErr),
			Invoke(func() { t.Errorf("Invoke should not be called") }),
		)
		err := app.Err()
		require.Error(t, err)
		assert.EqualError(t, err, "module failure")
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("MultipleErrorOption", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		errA := errors.New("module A failure")
		errB := errors.New("module B failure")
		app := NewForTest(t,
			Provide(func() A {
				t.Errorf("Provide should not be called")
				return A{}
			},
			),
			Invoke(func(A) { t.Errorf("Invoke should not be called") }),
			Error(
				errA,
				errB,
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, errA)
		assert.ErrorIs(t, err, errB)
		assert.NotContains(t, err.Error(), "not in the container")
	})

	t.Run("ProvideAndInvokeErrorsAreIgnored", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		app := NewForTest(t,
			Provide(func(b B) A {
				t.Errorf("B is missing from the container; Provide should not be called")
				return A{}
			},
			),
			Error(errors.New("module failure")),
			Invoke(func(A) { t.Errorf("A was not provided; Invoke should not be called") }),
		)
		err := app.Err()
		assert.EqualError(t, err, "module failure")
	})
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("OptionsComposition", func(t *testing.T) {
		t.Parallel()

		var n int
		construct := func() struct{} {
			n++
			return struct{}{}
		}
		use := func(struct{}) {
			n++
		}
		app := fxtest.New(t, Options(Provide(construct), Invoke(use)))
		defer app.RequireStart().RequireStop()
		assert.Equal(t, 2, n)
	})

	t.Run("ProvidesCalledInGraphOrder", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		initOrder := 0
		new1 := func() type1 {
			initOrder++
			assert.Equal(t, 1, initOrder)
			return type1{}
		}
		new2 := func(type1) type2 {
			initOrder++
			assert.Equal(t, 2, initOrder)
			return type2{}
		}
		new3 := func(type1, type2) type3 {
			initOrder++
			assert.Equal(t, 3, initOrder)
			return type3{}
		}
		biz := func(s1 type1, s2 type2, s3 type3) {
			initOrder++
			assert.Equal(t, 4, initOrder)
		}
		app := fxtest.New(t,
			Provide(new1, new2, new3),
			Invoke(biz),
		)
		defer app.RequireStart().RequireStop()
		assert.Equal(t, 4, initOrder)
	})

	t.Run("ProvidesCalledLazily", func(t *testing.T) {
		t.Parallel()

		count := 0
		newBuffer := func() *bytes.Buffer {
			t.Error("this module should not init: no provided type relies on it")
			return nil
		}
		newEmpty := func() struct{} {
			count++
			return struct{}{}
		}
		app := fxtest.New(t,
			Provide(newBuffer, newEmpty),
			Invoke(func(struct{}) { count++ }),
		)
		defer app.RequireStart().RequireStop()
		assert.Equal(t, 2, count)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		spy := new(fxlog.Spy)
		New(
			Provide(&bytes.Buffer{}), // error, not a constructor
			WithLogger(func() fxevent.Logger { return spy }),
		)
		require.Equal(t, []string{"Provided", "Provided", "Provided", "Provided", "LoggerInitialized"}, spy.EventTypes())
		// First 3 provides are Fx types (Lifecycle, Shutdowner, DotGraph).
		assert.Contains(t, spy.Events()[3].(*fxevent.Provided).Err.Error(), "must provide constructor function")
	})
}

func TestTimeoutOptions(t *testing.T) {
	t.Parallel()

	const timeout = time.Minute
	// Further assertions can't succeed unless the test timeout is greater than the default.
	require.True(t, timeout > DefaultTimeout, "test assertions require timeout greater than default")

	var started, stopped bool
	assertCustomContext := func(ctx context.Context, phase string) {
		deadline, ok := ctx.Deadline()
		if assert.True(t, ok, "no %s deadline", phase) {
			remaining := time.Until(deadline)
			assert.True(t, remaining > DefaultTimeout, "didn't respect customized %s timeout", phase)
		}
	}
	verify := func(lc Lifecycle) {
		lc.Append(Hook{
			OnStart: func(ctx context.Context) error {
				assertCustomContext(ctx, "start")
				started = true
				return nil
			},
			OnStop: func(ctx context.Context) error {
				assertCustomContext(ctx, "stop")
				stopped = true
				return nil
			},
		})
	}
	app := fxtest.New(
		t,
		Invoke(verify),
		StartTimeout(timeout),
		StopTimeout(timeout),
	)

	app.RequireStart().RequireStop()
	assert.True(t, started, "app wasn't started")
	assert.True(t, stopped, "app wasn't stopped")
}

func TestAppRunTimeout(t *testing.T) {
	t.Parallel()

	// Fails with an error immediately.
	failure := func() error {
		return errors.New("great sadness")
	}

	// Builds a hook that takes much longer than the application timeout.
	takeVeryLong := func(clock *fxclock.Mock) func() error {
		return func() error {
			// We'll exceed the start and stop timeouts,
			// and then some.
			for i := 0; i < 3; i++ {
				clock.Add(time.Second)
			}

			return errors.New("user should not see this")
		}
	}

	tests := []struct {
		desc string

		// buildHook builds and returns the hooks for this test case.
		buildHooks func(*fxclock.Mock) []Hook

		// Type of the fxevent we want.
		// Does not reflect the exact value.
		wantEventType fxevent.Event
	}{
		{
			// Timeout starting an application.
			desc: "OnStart timeout",
			buildHooks: func(clock *fxclock.Mock) []Hook {
				return []Hook{
					StartHook(takeVeryLong(clock)),
				}
			},
			wantEventType: &fxevent.Started{},
		},
		{
			// Timeout during a rollback because start failed.
			desc: "rollback timeout",
			buildHooks: func(clock *fxclock.Mock) []Hook {
				return []Hook{
					// The hooks are separate because
					// OnStop will not be run if that hook failed.
					StartHook(takeVeryLong(clock)),
					StopHook(failure),
				}
			},
			wantEventType: &fxevent.Started{},
		},
		{
			// Timeout during a stop.
			desc: "OnStop timeout",
			buildHooks: func(clock *fxclock.Mock) []Hook {
				return []Hook{
					StopHook(takeVeryLong(clock)),
				}
			},
			wantEventType: &fxevent.Stopped{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			mockClock := fxclock.NewMock()

			var (
				exitCode int
				exited   bool
			)
			exit := func(code int) {
				exited = true
				exitCode = code
			}
			defer func() {
				assert.True(t, exited,
					"os.Exit must be called")
			}()

			// If the OnStart hook for this is invoked,
			// it means that the Start did not fail.
			// In that case, shut down immediately
			// rather than block forever.
			shutdown := func(sd Shutdowner, lc Lifecycle) {
				lc.Append(Hook{
					OnStart: func(context.Context) error {
						return sd.Shutdown()
					},
				})
			}

			app, spy := NewSpied(
				StartTimeout(time.Second),
				StopTimeout(time.Second),
				WithExit(exit),
				WithClock(mockClock),
				Invoke(func(lc Lifecycle) {
					hooks := tt.buildHooks(mockClock)
					for _, h := range hooks {
						lc.Append(h)
					}
				}),
				Invoke(shutdown),
			)

			app.Run()
			assert.NotZero(t, exitCode,
				"exit code mismatch")

			eventType := reflect.TypeOf(tt.wantEventType).Elem().Name()
			matchingEvents := spy.Events().SelectByTypeName(eventType)
			require.Len(t, matchingEvents, 1,
				"expected a %q event", eventType)

			event := matchingEvents[0]
			errv := reflect.ValueOf(event).Elem().FieldByName("Err")
			require.True(t, errv.IsValid(),
				"event %q does not have an Err attribute", eventType)

			err, _ := errv.Interface().(error)
			assert.ErrorIs(t, err, context.DeadlineExceeded,
				"should fail because of a timeout")
		})
	}
}

func TestAppStart(t *testing.T) {
	t.Parallel()

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()

		mockClock := fxclock.NewMock()

		type A struct{}
		blocker := func(lc Lifecycle) *A {
			lc.Append(
				Hook{
					OnStart: func(ctx context.Context) error {
						mockClock.Add(5 * time.Second)
						return ctx.Err()
					},
				},
			)
			return &A{}
		}
		// NOTE: for tests that gets cancelled/times out during lifecycle methods, it's possible
		// for them to run into race with fxevent logs getting written to testing.T with the
		// remainder of the tests. As a workaround, we provide fxlog.Spy to prevent the lifecycle
		// goroutine from writing to testing.T.
		spy := new(fxlog.Spy)
		app := NewForTest(t,
			WithLogger(func() fxevent.Logger { return spy }),
			WithClock(mockClock),
			Provide(blocker),
			Invoke(func(*A) {}),
		)

		ctx, cancel := mockClock.WithTimeout(context.Background(), time.Second)

		err := app.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, "context deadline exceeded", err.Error())
		cancel()
	})

	t.Run("TimeoutWithFinishedHooks", func(t *testing.T) {
		t.Parallel()

		mockClock := fxclock.NewMock()

		type A struct{}
		type B struct{ A *A }
		type C struct{ B *B }
		newA := func(lc Lifecycle) *A {
			lc.Append(
				Hook{
					OnStart: func(context.Context) error {
						mockClock.Add(100 * time.Millisecond)
						return nil
					},
				},
			)
			return &A{}
		}
		newB := func(lc Lifecycle, a *A) *B {
			lc.Append(
				Hook{
					OnStart: func(context.Context) error {
						mockClock.Add(300 * time.Millisecond)
						return nil
					},
				},
			)
			return &B{a}
		}
		newC := func(lc Lifecycle, b *B) *C {
			lc.Append(
				Hook{
					OnStart: func(ctx context.Context) error {
						mockClock.Add(5 * time.Second)
						return ctx.Err()
					},
				},
			)
			return &C{b}
		}

		// NOTE: for tests that gets cancelled/times out during lifecycle methods, it's possible
		// for them to run into race with fxevent logs getting written to testing.T with the
		// remainder of the tests. As a workaround, we provide fxlog.Spy to prevent the lifecycle
		// goroutine from writing to testing.T.
		spy := new(fxlog.Spy)
		app := New(
			WithLogger(func() fxevent.Logger { return spy }),
			WithClock(mockClock),
			Provide(newA, newB, newC),
			Invoke(func(*C) {}),
		)

		ctx, cancel := mockClock.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := app.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, "context deadline exceeded", err.Error())
	})

	t.Run("CtxCancelledDuringStart", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		running := make(chan struct{})
		newA := func(lc Lifecycle) *A {
			lc.Append(
				Hook{
					OnStart: func(ctx context.Context) error {
						close(running)
						<-ctx.Done()
						return ctx.Err()
					},
				},
			)
			return &A{}
		}

		// NOTE: for tests that gets cancelled/times out during lifecycle methods, it's possible
		// for them to run into race with fxevent logs getting written to testing.T with the
		// remainder of the tests. As a workaround, we provide fxlog.Spy to prevent the lifecycle
		// goroutine from writing to testing.T.
		spy := new(fxlog.Spy)
		app := New(
			WithLogger(func() fxevent.Logger { return spy }),
			Provide(newA),
			Invoke(func(*A) {}),
		)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-running
			cancel()
		}()
		err := app.Start(ctx)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "context deadline exceeded")
		assert.NotContains(t, err.Error(), "timed out while executing hook OnStart")
	})

	t.Run("race test", func(t *testing.T) {
		t.Parallel()

		secondStart := make(chan struct{}, 1)
		firstStop := make(chan struct{}, 1)
		startReturn := make(chan struct{}, 1)

		var stop1Run bool
		app := New(
			Invoke(func(lc Lifecycle) {
				lc.Append(Hook{
					OnStart: func(context.Context) error {
						return nil
					},
					OnStop: func(context.Context) error {
						if stop1Run {
							require.Fail(t, "Hooks should only run once")
						}
						stop1Run = true
						close(firstStop)
						<-startReturn
						return nil
					},
				})
				lc.Append(Hook{
					OnStart: func(context.Context) error {
						close(secondStart)
						<-firstStop
						return nil
					},
					OnStop: func(context.Context) error {
						assert.Fail(t, "Stop hook 2 should not be called "+
							"if start hook 2 does not finish")
						return nil
					},
				})
			}),
		)

		go func() {
			app.Start(context.Background())
			close(startReturn)
		}()

		<-secondStart
		app.Stop(context.Background())
		assert.True(t, stop1Run)
	})

	t.Run("CtxTimeoutDuringStartStillRunsStopHooks", func(t *testing.T) {
		t.Parallel()

		var ran bool
		mockClock := fxclock.NewMock()
		app := New(
			WithClock(mockClock),
			Invoke(func(lc Lifecycle) {
				lc.Append(Hook{
					OnStart: func(ctx context.Context) error {
						return nil
					},
					OnStop: func(ctx context.Context) error {
						ran = true
						return nil
					},
				})
				lc.Append(Hook{
					OnStart: func(ctx context.Context) error {
						mockClock.Add(5 * time.Second)
						return ctx.Err()
					},
					OnStop: func(ctx context.Context) error {
						assert.Fail(t, "This Stop hook should not be called")
						return nil
					},
				})
			}),
		)

		startCtx, cancelStart := mockClock.WithTimeout(context.Background(), time.Second)
		defer cancelStart()
		err := app.Start(startCtx)
		require.Error(t, err)
		assert.ErrorContains(t, err, "context deadline exceeded")
		require.NoError(t, app.Stop(context.Background()))
		assert.True(t, ran, "Stop hook for the Start hook that finished running should have been called")
	})

	t.Run("Rollback", func(t *testing.T) {
		t.Parallel()

		failStart := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStart: func(context.Context) error {
				return errors.New("OnStart fail")
			}})
			return struct{}{}
		}
		app, spy := NewSpied(
			Provide(failStart),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStart fail")

		assert.Equal(t, []string{
			"Provided", "Provided", "Provided", "Provided",
			"LoggerInitialized",
			"Invoking",
			"Run",
			"Run",
			"Invoked",
			"OnStartExecuting", "OnStartExecuted",
			"RollingBack",
			"RolledBack",
			"Started",
		}, spy.EventTypes())
	})

	t.Run("StartAndStopErrors", func(t *testing.T) {
		t.Parallel()

		errStop1 := errors.New("OnStop fail 1")
		errStart2 := errors.New("OnStart fail 2")
		fail := func(lc Lifecycle) struct{} {
			lc.Append(Hook{
				OnStart: func(context.Context) error { return nil },
				OnStop:  func(context.Context) error { return errStop1 },
			})
			lc.Append(Hook{
				OnStart: func(context.Context) error { return errStart2 },
				OnStop:  func(context.Context) error { assert.Fail(t, "should be never called"); return nil },
			})
			return struct{}{}
		}
		app, spy := NewSpied(
			Provide(fail),
			Invoke(func(struct{}) {}),
		)
		err := app.Start(context.Background())
		require.Error(t, err)
		assert.Equal(t, []error{errStart2, errStop1}, multierr.Errors(err))

		assert.Equal(t, []string{
			"Provided", "Provided", "Provided", "Provided",
			"LoggerInitialized",
			"Invoking",
			"Run",
			"Run",
			"Invoked",
			"OnStartExecuting", "OnStartExecuted",
			"OnStartExecuting", "OnStartExecuted",
			"RollingBack",
			"OnStopExecuting", "OnStopExecuted",
			"RolledBack",
			"Started",
		}, spy.EventTypes())
	})

	t.Run("InvokeNonFunction", func(t *testing.T) {
		t.Parallel()

		spy := new(fxlog.Spy)

		app := New(WithLogger(func() fxevent.Logger { return spy }), Invoke(struct{}{}))
		err := app.Err()
		require.Error(t, err, "expected start failure")
		assert.Contains(t, err.Error(), "can't invoke non-function")

		// Example
		// fx.Invoke({}) called from:
		// go.uber.org/fx_test.TestAppStart.func4
		//         /.../fx/app_test.go:525
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		// Failed: can't invoke non-function {} (type struct {})
		require.Equal(t,
			[]string{"Provided", "Provided", "Provided", "LoggerInitialized", "Invoking", "Invoked"},
			spy.EventTypes())
		failedEvent := spy.Events()[len(spy.EventTypes())-1].(*fxevent.Invoked)
		assert.Contains(t, failedEvent.Err.Error(), "can't invoke non-function")
		assert.Contains(t, failedEvent.Trace, "go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, failedEvent.Trace, "/app_test.go")
	})

	t.Run("ProvidingAProvideShouldFail", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		app := NewForTest(t,
			Provide(
				func() type1 { return type1{} },
				Provide(
					func() type2 { return type2{} },
					func() type3 { return type3{} },
				),
			),
		)

		err := app.Err()
		require.Error(t, err, "expected start failure")

		// Example:
		// fx.Option should be passed to fx.New directly, not to fx.Provide: fx.Provide received fx.Provide(go.uber.org/fx_test.TestAppStart.func5.2(), go.uber.org/fx_test.TestAppStart.func5.3()) from:
		// go.uber.org/fx_test.TestAppStart.func5
		//         /.../fx/app_test.go:550
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Provide")
		assert.Contains(t, err.Error(), "fx.Provide received fx.Provide(go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "/app_test.go")
	})

	t.Run("InvokingAnInvokeShouldFail", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}

		app := NewForTest(t,
			Provide(func() type1 { return type1{} }),
			Invoke(Invoke(func(type1) {
			})),
		)
		newErr := app.Err()
		require.Error(t, newErr)

		err := app.Start(context.Background())
		require.Error(t, err, "expected start failure")
		assert.Equal(t, err, newErr, "start should return the same error fx.New encountered")

		// Example
		// fx.Option should be passed to fx.New directly, not to fx.Invoke: fx.Invoke received fx.Invoke(go.uber.org/fx_test.TestAppStart.func6.2()) from:
		// go.uber.org/fx_test.TestAppStart.func6
		//         /.../fx/app_test.go:579
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Invoke")
		assert.Contains(t, err.Error(), "fx.Invoke received fx.Invoke(go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "/app_test.go")
	})

	t.Run("ProvidingOptionsShouldFail", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		type type2 struct{}
		type type3 struct{}

		module := Options(
			Provide(
				func() type1 { return type1{} },
				func() type2 { return type2{} },
			),
			Invoke(func(type1) {
				require.FailNow(t, "module Invoked must not be called")
			}),
		)

		app := NewForTest(t,
			Provide(
				func() type3 { return type3{} },
				module,
			),
		)
		err := app.Err()
		require.Error(t, err, "expected start failure")

		// Example:
		// fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor: fx.Provide received go.uber.org/fx_test.TestAnnotatedWrongUsage.func2.1() from:
		// go.uber.org/fx_test.TestAnnotatedWrongUsage.func2
		//         /.../fx/annotated_test.go:76
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Option should be passed to fx.New directly, not to fx.Provide")
		assert.Contains(t, err.Error(), "fx.Provide received fx.Options(fx.Provide(go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestAppStart")
		assert.Contains(t, err.Error(), "/app_test.go")
	})

	t.Run("HookGoroutineExitsErrorMsg", func(t *testing.T) {
		t.Parallel()

		addHook := func(lc Lifecycle) {
			lc.Append(Hook{
				OnStart: func(ctx context.Context) error {
					runtime.Goexit()
					return nil
				},
			})
		}
		app := fxtest.New(t,
			Invoke(addHook),
		)
		err := app.Start(context.Background()).Error()
		assert.Contains(t, "goroutine exited without returning", err)
	})

	t.Run("StartTwiceWithHooksErrors", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app := fxtest.New(t,
			Invoke(func(lc Lifecycle) {
				lc.Append(Hook{
					OnStart: func(ctx context.Context) error { return nil },
					OnStop:  func(ctx context.Context) error { return nil },
				})
			}),
		)
		assert.NoError(t, app.Start(ctx))
		err := app.Start(ctx)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, "attempted to start lifecycle when in state: started")
		}
		app.Stop(ctx)
		assert.NoError(t, app.Start(ctx))
		app.Stop(ctx)
	})
}

func TestAppStop(t *testing.T) {
	t.Parallel()

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()

		mockClock := fxclock.NewMock()

		block := func(ctx context.Context) error {
			mockClock.Add(5 * time.Second)
			return ctx.Err()
		}
		// NOTE: for tests that gets cancelled/times out during lifecycle methods, it's possible
		// for them to run into race with fxevent logs getting written to testing.T with the
		// remainder of the tests. As a workaround, we provide fxlog.Spy to prevent the lifecycle
		// goroutine from writing to testing.T.
		spy := new(fxlog.Spy)
		app := New(Invoke(func(l Lifecycle) { l.Append(Hook{OnStop: block}) }),
			WithLogger(func() fxevent.Logger { return spy }),
			WithClock(mockClock),
		)

		err := app.Start(context.Background())
		require.Nil(t, err)

		ctx, cancel := mockClock.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err = app.Stop(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("StopError", func(t *testing.T) {
		t.Parallel()

		failStop := func(lc Lifecycle) struct{} {
			lc.Append(Hook{OnStop: func(context.Context) error {
				return errors.New("OnStop fail")
			}})
			return struct{}{}
		}
		app := fxtest.New(t,
			Provide(failStop),
			Invoke(func(struct{}) {}),
		)
		app.RequireStart()
		err := app.Stop(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnStop fail")
	})
}

func TestValidateApp(t *testing.T) {
	t.Parallel()

	// helper to use the test logger
	validateApp := func(t *testing.T, opts ...Option) error {
		return ValidateApp(
			append(opts, Logger(fxtest.NewTestPrinter(t)))...,
		)
	}

	t.Run("do not run provides on graph validation", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		err := validateApp(t,
			Provide(func() *type1 {
				t.Error("provide must not be called")
				return nil
			}),
			Invoke(func(*type1) {}),
		)
		require.NoError(t, err)
	})
	t.Run("do not run provides nor invokes on graph validation", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		err := validateApp(t,
			Provide(func() *type1 {
				t.Error("provide must not be called")
				return nil
			}),
			Invoke(func(*type1) {
				t.Error("invoke must not be called")
			}),
		)
		require.NoError(t, err)
	})
	t.Run("provide depends on something not available", func(t *testing.T) {
		t.Parallel()

		type type1 struct{}
		err := validateApp(t,
			Provide(func(type1) int { return 0 }),
			Invoke(func(int) error { return nil }),
		)
		require.Error(t, err, "fx.ValidateApp should error on argument not available")
		errMsg := err.Error()
		assert.Contains(t, errMsg, "could not build arguments for function")
		assert.Contains(t, errMsg, "failed to build int: missing dependencies for function")
		assert.Contains(t, errMsg, "missing type: fx_test.type1")
	})
	t.Run("provide introduces a cycle", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}
		err := validateApp(t,
			Provide(func(A) B { return B{} }),
			Provide(func(B) A { return A{} }),
			Invoke(func(B) {}),
		)
		require.Error(t, err, "fx.ValidateApp should error on cycle")
		errMsg := err.Error()
		assert.Contains(t, errMsg, "cycle detected in dependency graph")
	})
	t.Run("invoke a type that's not available", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		err := validateApp(t,
			Invoke(func(A) {}),
		)
		require.Error(t, err, "fx.ValidateApp should return an error on missing invoke dep")
		errMsg := err.Error()
		assert.Contains(t, errMsg, "missing dependencies for function")
		assert.Contains(t, errMsg, "missing type: fx_test.A")
	})
	t.Run("no error", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		err := validateApp(t,
			Provide(func() A {
				return A{}
			}),
			Invoke(func(A) {}),
		)
		require.NoError(t, err, "fx.ValidateApp should not return an error")
	})
}

func TestHookConstructors(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		var (
			calls = map[string]int{
				"start func":         0,
				"start func err":     0,
				"start ctx func":     0,
				"start ctx func err": 0,
				"stop func":          0,
				"stop func err":      0,
				"stop ctx func":      0,
				"stop ctx func err":  0,
			}
			nilFunc             func()
			nilErrorFunc        func() error
			nilContextFunc      func(context.Context)
			nilContextErrorFunc func(context.Context) error
		)

		fxtest.New(
			t,
			Invoke(func(lc Lifecycle) {
				// Nil functions
				lc.Append(StartStopHook(nilFunc, nilErrorFunc))
				lc.Append(StartStopHook(nilContextFunc, nilContextErrorFunc))

				// Start hooks
				lc.Append(StartHook(func() {
					calls["start func"]++
				}))
				lc.Append(StartHook(func() error {
					calls["start func err"]++
					return nil
				}))
				lc.Append(StartHook(func(context.Context) {
					calls["start ctx func"]++
				}))
				lc.Append(StartHook(func(context.Context) error {
					calls["start ctx func err"]++
					return nil
				}))

				// Stop hooks
				lc.Append(StopHook(func() {
					calls["stop func"]++
				}))
				lc.Append(StopHook(func() error {
					calls["stop func err"]++
					return nil
				}))
				lc.Append(StopHook(func(context.Context) {
					calls["stop ctx func"]++
				}))
				lc.Append(StopHook(func(context.Context) error {
					calls["stop ctx func err"]++
					return nil
				}))

				// StartStop hooks
				lc.Append(StartStopHook(
					func() {
						calls["start func"]++
					},
					func() error {
						calls["stop func err"]++
						return nil
					},
				))
				lc.Append(StartStopHook(
					func() error {
						calls["start func err"]++
						return nil
					},
					func(context.Context) {
						calls["stop ctx func"]++
					},
				))
				lc.Append(StartStopHook(
					func(context.Context) {
						calls["start ctx func"]++
					},
					func(context.Context) error {
						calls["stop ctx func err"]++
						return nil
					},
				))
				lc.Append(StartStopHook(
					func(context.Context) error {
						calls["start ctx func err"]++
						return nil
					},
					func() {
						calls["stop func"]++
					},
				))
			}),
		).RequireStart().RequireStop()

		for name, count := range calls {
			require.Equalf(t, 2, count, "bad call count: %s (%d)", name, count)
		}
	})

	t.Run("start errors", func(t *testing.T) {
		wantErr := errors.New("oh no")

		// Verify that wrapped `func() error` funcs produce the expected error.
		err := New(Invoke(func(lc Lifecycle) {
			lc.Append(StartHook(func() error {
				return wantErr
			}))
		})).Start(context.Background())
		require.ErrorContains(t, err, wantErr.Error())

		// Verify that wrapped `func(context.Context) error` funcs produce the
		// expected error.
		err = New(Invoke(func(lc Lifecycle) {
			lc.Append(StartHook(func(ctx context.Context) error {
				return wantErr
			}))
		})).Start(context.Background())
		require.ErrorContains(t, err, wantErr.Error())
	})

	t.Run("start deadline", func(t *testing.T) {
		wantCtx, cancel := context.WithTimeout(context.Background(), 123*time.Second)
		defer cancel()

		// Verify that `func(context.Context)` funcs receive the expected
		// deadline.
		app := New(Invoke(func(lc Lifecycle) {
			lc.Append(StartHook(func(ctx context.Context) {
				var (
					want, wantOK = wantCtx.Deadline()
					give, giveOK = ctx.Deadline()
				)

				require.Equal(t, wantOK, giveOK)
				require.True(t, want.Equal(give))
			}))
		}))
		require.NoError(t, app.Start(wantCtx))
		require.NoError(t, app.Stop(wantCtx))

		// Verify that `func(context.Context) error` funcs receive the expected
		// deadline.
		app = New(Invoke(func(lc Lifecycle) {
			lc.Append(StartHook(func(ctx context.Context) error {
				var (
					want, wantOK = wantCtx.Deadline()
					give, giveOK = ctx.Deadline()
				)

				require.Equal(t, wantOK, giveOK)
				require.True(t, want.Equal(give))

				return nil
			}))
		}))
		require.NoError(t, app.Start(wantCtx))
		require.NoError(t, app.Stop(wantCtx))
	})

	t.Run("stop errors", func(t *testing.T) {
		var (
			ctx     = context.Background()
			wantErr = errors.New("oh no")
		)

		// Verify that wrapped `func() error` funcs produce the expected error.
		app := New(Invoke(func(lc Lifecycle) {
			lc.Append(StopHook(func() error {
				return wantErr
			}))
		}))
		require.NoError(t, app.Start(ctx))
		require.ErrorContains(t, app.Stop(ctx), wantErr.Error())

		// Verify that wrapped `func(context.Context) error` funcs produce the
		// expected error.
		app = New(Invoke(func(lc Lifecycle) {
			lc.Append(StopHook(func(ctx context.Context) error {
				return wantErr
			}))
		}))
		require.NoError(t, app.Start(ctx))
		require.ErrorIs(t, app.Stop(ctx), wantErr)
	})

	t.Run("stop deadline", func(t *testing.T) {
		wantCtx, cancel := context.WithTimeout(
			context.Background(),
			123*time.Second,
		)
		defer cancel()

		// Verify that `func(context.Context)` funcs receive the expected
		// deadline.
		app := New(Invoke(func(lc Lifecycle) {
			lc.Append(StopHook(func(ctx context.Context) {
				var (
					want, wantOK = wantCtx.Deadline()
					give, giveOK = ctx.Deadline()
				)

				require.Equal(t, wantOK, giveOK)
				require.True(t, want.Equal(give))
			}))
		}))
		require.NoError(t, app.Start(wantCtx))
		require.NoError(t, app.Stop(wantCtx))

		// Verify that `func(context.Context) error` funcs receive the expected
		// deadline.
		app = New(Invoke(func(lc Lifecycle) {
			lc.Append(StopHook(func(ctx context.Context) error {
				var (
					want, wantOK = wantCtx.Deadline()
					give, giveOK = ctx.Deadline()
				)

				require.Equal(t, wantOK, giveOK)
				require.True(t, want.Equal(give))

				return nil
			}))
		}))
		require.NoError(t, app.Start(wantCtx))
		require.NoError(t, app.Stop(wantCtx))
	})

	t.Run("typed", func(t *testing.T) {
		type (
			funcType       func()
			funcErrType    func() error
			ctxFuncType    func(context.Context)
			ctxFuncErrType func(context.Context) error
		)

		var (
			calls  int
			myFunc = funcType(func() {
				calls++
			})
			myFuncErr = funcErrType(func() error {
				calls++
				return nil
			})
			myCtxFunc = ctxFuncType(func(context.Context) {
				calls++
			})
			myCtxFuncErr = ctxFuncErrType(func(context.Context) error {
				calls++
				return nil
			})
		)

		// Ensure that user function types that are assignable to supported
		// base function types are converted as expected.
		require.NotPanics(t, func() {
			fxtest.New(t, Invoke(func(lc Lifecycle) {
				lc.Append(StartStopHook(myFunc, myFuncErr))
				lc.Append(StartStopHook(myCtxFunc, myCtxFuncErr))
			})).RequireStart().RequireStop()
		})
		require.Equal(t, 4, calls)
	})
}

func TestDone(t *testing.T) {
	t.Parallel()

	app := fxtest.New(t)
	defer app.RequireStop()
	done := app.Done()
	require.NotNil(t, done, "Got a nil channel.")
	select {
	case sig := <-done:
		t.Fatalf("Got unexpected signal %v from application's Done channel.", sig)
	default:
	}
}

// TestShutdownThenWait tests that if we call .Shutdown before waiting, the wait
// will still return the last shutdown signal.
func TestShutdownThenWait(t *testing.T) {
	t.Parallel()

	var (
		s       Shutdowner
		stopped bool
	)
	app := fxtest.New(
		t,
		Populate(&s),
		Invoke(func(lc Lifecycle) {
			lc.Append(StopHook(func() {
				stopped = true
			}))
		}),
	).RequireStart()
	require.NotNil(t, s)

	err := s.Shutdown(ExitCode(1337))
	assert.NoError(t, err)
	assert.False(t, stopped)

	shutdownSig := <-app.Wait()
	assert.Equal(t, 1337, shutdownSig.ExitCode)
	assert.False(t, stopped)

	app.RequireStop()
	assert.True(t, stopped)
}

func TestReplaceLogger(t *testing.T) {
	t.Parallel()

	spy := new(fxlog.Spy)
	app := fxtest.New(t, WithLogger(func() fxevent.Logger { return spy }))
	app.RequireStart().RequireStop()
	assert.Equal(t, []string{
		"Provided",
		"Provided",
		"Provided",
		"LoggerInitialized",
		"Started",
		"Stopped",
	}, spy.EventTypes())
}

func TestNopLogger(t *testing.T) {
	t.Parallel()

	app := fxtest.New(t, NopLogger)
	app.RequireStart().RequireStop()
}

func TestCustomLoggerWithPrinter(t *testing.T) {
	t.Parallel()

	// If we provide both, an fx.Logger and fx.WithLogger, and the logger
	// fails, we should fall back to the fx.Logger.

	var buff bytes.Buffer
	app := New(
		Logger(log.New(&buff, "", 0)),
		WithLogger(func() (fxevent.Logger, error) {
			return nil, errors.New("great sadness")
		}),
	)
	err := app.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "great sadness")

	out := buff.String()
	assert.Contains(t, out, "failed to build fxevent.Logger")
	assert.Contains(t, out, "great sadness")
}

func TestCustomLoggerWithLifecycle(t *testing.T) {
	t.Parallel()

	var started, stopped bool
	defer func() {
		assert.True(t, started, "never started")
		assert.True(t, stopped, "never stopped")
	}()

	var buff bytes.Buffer
	defer func() {
		assert.Empty(t, buff.String(), "unexpectedly wrote to the fallback logger")
	}()

	var spy fxlog.Spy
	app := New(
		// We expect WithLogger to do its job. This means we shouldn't
		// print anything to this buffer.
		Logger(log.New(&buff, "", 0)),
		WithLogger(func(lc Lifecycle) fxevent.Logger {
			lc.Append(Hook{
				OnStart: func(context.Context) error {
					assert.False(t, started, "started twice")
					started = true
					return nil
				},
				OnStop: func(context.Context) error {
					assert.False(t, stopped, "stopped twice")
					stopped = true
					return nil
				},
			})
			return &spy
		}),
	)

	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, app.Stop(context.Background()))

	assert.Equal(t, []string{
		"Provided",
		"Provided",
		"Provided",
		"Run",
		"LoggerInitialized",
		"OnStartExecuting", "OnStartExecuted",
		"Started",
		"OnStopExecuting", "OnStopExecuted",
		"Stopped",
	}, spy.EventTypes())
}

func TestCustomLoggerFailure(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	app := New(
		// We expect WithLogger to fail, so this buffer should be
		// contain information about the failure.
		Logger(log.New(&buff, "", 0)),
		WithLogger(func() (fxevent.Logger, error) {
			return nil, errors.New("great sadness")
		}),
	)
	require.Error(t, app.Err())

	out := buff.String()
	assert.Contains(t, out, "Failed to initialize custom logger")
	assert.Contains(t, out, "failed to build fxevent.Logger")
	assert.Contains(t, out, "received non-nil error from function")
	assert.Contains(t, out, "great sadness")
}

type testErrorWithGraph struct {
	graph string
}

func (we testErrorWithGraph) Graph() DotGraph {
	return DotGraph(we.graph)
}

func (we testErrorWithGraph) Error() string {
	return "great sadness"
}

func TestVisualizeError(t *testing.T) {
	t.Parallel()

	t.Run("NotWrappedError", func(t *testing.T) {
		t.Parallel()

		_, err := VisualizeError(errors.New("great sadness"))
		require.Error(t, err)
	})

	t.Run("WrappedErrorWithEmptyGraph", func(t *testing.T) {
		t.Parallel()

		graph, err := VisualizeError(testErrorWithGraph{graph: ""})
		assert.Empty(t, graph)
		require.Error(t, err)
	})

	t.Run("WrappedError", func(t *testing.T) {
		t.Parallel()

		graph, err := VisualizeError(testErrorWithGraph{graph: "graph"})
		assert.Equal(t, "graph", graph)
		require.NoError(t, err)
	})
}

func TestErrorHook(t *testing.T) {
	t.Parallel()

	t.Run("UnvisualizableError", func(t *testing.T) {
		t.Parallel()

		type A struct{}

		var graphErr error
		h := errHandlerFunc(func(err error) {
			_, graphErr = VisualizeError(err)
		})
		NewForTest(t,
			Provide(func() A { return A{} }),
			Invoke(func(A) error { return errors.New("great sadness") }),
			ErrorHook(h),
		)
		assert.Equal(t, errors.New("unable to visualize error"), graphErr)
	})

	t.Run("GraphWithError", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		var errStr, graphStr string
		h := errHandlerFunc(func(err error) {
			errStr = err.Error()
			graphStr, _ = VisualizeError(err)
		})
		NewForTest(t,
			Provide(func() (B, error) { return B{}, fmt.Errorf("great sadness") }),
			Provide(func(B) A { return A{} }),
			Invoke(func(A) {}),
			ErrorHook(&h),
		)
		assert.Contains(t, errStr, "great sadness")
		assert.Contains(t, graphStr, `"fx_test.B" [color=red];`)
		assert.Contains(t, graphStr, `"fx_test.A" [color=orange];`)
	})

	t.Run("GraphWithErrorInModule", func(t *testing.T) {
		t.Parallel()

		type A struct{}
		type B struct{}

		var errStr, graphStr string
		h := errHandlerFunc(func(err error) {
			errStr = err.Error()
			graphStr, _ = VisualizeError(err)
		})
		NewForTest(t,
			Module("module",
				Provide(func() (B, error) { return B{}, fmt.Errorf("great sadness") }),
				Provide(func(B) A { return A{} }),
				Invoke(func(A) {}),
				ErrorHook(&h),
			),
		)
		assert.Contains(t, errStr, "great sadness")
		assert.Contains(t, graphStr, `"fx_test.B" [color=red];`)
		assert.Contains(t, graphStr, `"fx_test.A" [color=orange];`)
	})
}

func TestOptionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give Option
		want string
	}{
		{
			desc: "Provide",
			give: Provide(bytes.NewReader),
			want: "fx.Provide(bytes.NewReader())",
		},
		{
			desc: "Invoked",
			give: Invoke(func(c io.Closer) error {
				return c.Close()
			}),
			want: "fx.Invoke(go.uber.org/fx_test.TestOptionString.func1())",
		},
		{
			desc: "Error/single",
			give: Error(errors.New("great sadness")),
			want: "fx.Error(great sadness)",
		},
		{
			desc: "Error/multiple",
			give: Error(errors.New("foo"), errors.New("bar")),
			want: "fx.Error(foo; bar)",
		},
		{
			desc: "Options/single",
			give: Options(Provide(bytes.NewBuffer)),
			// NOTE: We don't prune away fx.Options for the empty
			// case because we want to attach additional
			// information to the fx.Options object in the future.
			want: "fx.Options(fx.Provide(bytes.NewBuffer()))",
		},
		{
			desc: "Options/multiple",
			give: Options(
				Provide(bytes.NewBufferString),
				Invoke(func(buf *bytes.Buffer) {
					buf.WriteString("hello")
				}),
			),
			want: "fx.Options(" +
				"fx.Provide(bytes.NewBufferString()), " +
				"fx.Invoke(go.uber.org/fx_test.TestOptionString.func2())" +
				")",
		},
		{
			desc: "StartTimeout",
			give: StartTimeout(time.Second),
			want: "fx.StartTimeout(1s)",
		},
		{
			desc: "StopTimeout",
			give: StopTimeout(5 * time.Second),
			want: "fx.StopTimeout(5s)",
		},
		{
			desc: "RecoverFromPanics",
			give: RecoverFromPanics(),
			want: "fx.RecoverFromPanics()",
		},
		{
			desc: "Logger",
			give: WithLogger(func() fxevent.Logger { return testLogger{t} }),
			want: "fx.WithLogger(go.uber.org/fx_test.TestOptionString.func3())",
		},
		{
			desc: "ErrorHook",
			give: ErrorHook(testErrorHandler{t}),
			want: "fx.ErrorHook(TestOptionString)",
		},
		{
			desc: "Supplied/simple",
			give: Supply(bytes.NewReader(nil), bytes.NewBuffer(nil)),
			want: "fx.Supply(*bytes.Reader, *bytes.Buffer)",
		},
		{
			desc: "Supplied/Annotated",
			give: Supply(Annotated{Target: bytes.NewReader(nil)}),
			want: "fx.Supply(*bytes.Reader)",
		},
		{
			desc: "Decorate",
			give: Decorate(bytes.NewBufferString),
			want: "fx.Decorate(bytes.NewBufferString())",
		},
		{
			desc: "Replace",
			give: Replace(bytes.NewReader(nil)),
			want: "fx.Replace(*bytes.Reader)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			stringer, ok := tt.give.(fmt.Stringer)
			require.True(t, ok, "option must implement stringer")
			assert.Equal(t, tt.want, stringer.String())
		})
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type testLogger struct{ t *testing.T }

func (l testLogger) Printf(s string, args ...interface{}) {
	l.t.Logf(s, args...)
}

func (l testLogger) LogEvent(event fxevent.Event) {
	l.t.Logf("emitted event %#v", event)
}

func (l testLogger) String() string {
	return l.t.Name()
}

type testErrorHandler struct{ t *testing.T }

func (h testErrorHandler) HandleError(err error) {
	h.t.Error(err)
}

func (h testErrorHandler) String() string {
	return h.t.Name()
}
