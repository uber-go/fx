// Copyright (c) 2019-2021 Uber Technologies, Inc.
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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
)

func TestAnnotated(t *testing.T) {
	t.Parallel()

	type a struct {
		name string
	}
	type in struct {
		fx.In

		A *a `name:"foo"`
	}
	newA := func() *a {
		return &a{name: "foo"}
	}
	t.Run("Provide", func(t *testing.T) {
		t.Parallel()

		var in in
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
			),
			fx.Populate(&in),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, in.A, "expected in.A to be injected")
		assert.Equal(t, "foo", in.A.name, "expected to get a type 'a' of name 'foo'")
	})
}

type asStringer struct {
	name string
}

func (as *asStringer) String() string {
	return as.name
}

type anotherStringer struct {
	name string
}

func (a anotherStringer) String() string {
	return a.name
}

func TestAnnotatedAs(t *testing.T) {
	t.Parallel()
	type in struct {
		fx.In

		S fmt.Stringer `name:"goodStringer"`
	}
	type myStringer interface {
		String() string
	}

	newAsStringer := func() *asStringer {
		return &asStringer{
			name: "a good stringer",
		}
	}

	tests := []struct {
		desc    string
		provide fx.Option
		invoke  interface{}
	}{
		{
			desc: "provide a good stringer",
			provide: fx.Provide(
				fx.Annotate(newAsStringer, fx.As(new(fmt.Stringer))),
			),
			invoke: func(s fmt.Stringer) {
				assert.Equal(t, s.String(), "a good stringer")
			},
		},
		{
			desc: "value type implementing interface",
			provide: fx.Provide(
				fx.Annotate(func() anotherStringer {
					return anotherStringer{
						"another stringer",
					}
				}, fx.As(new(fmt.Stringer))),
			),
			invoke: func(s fmt.Stringer) {
				assert.Equal(t, s.String(), "another stringer")
			},
		},
		{
			desc: "provide with multiple types As",
			provide: fx.Provide(fx.Annotate(func() (*asStringer, *bytes.Buffer) {
				buf := make([]byte, 1)
				b := bytes.NewBuffer(buf)
				return &asStringer{name: "stringer"}, b
			}, fx.As(new(fmt.Stringer), new(io.Writer)))),
			invoke: func(s fmt.Stringer, w io.Writer) {
				w.Write([]byte(s.String()))
			},
		},
		{
			desc: "provide as with result annotation",
			provide: fx.Provide(
				fx.Annotate(func() *asStringer {
					return &asStringer{name: "stringer"}
				},
					fx.ResultTags(`name:"goodStringer"`),
					fx.As(new(fmt.Stringer))),
			),
			invoke: func(i in) {
				assert.Equal(t, "stringer", i.S.String())
			},
		},
		{
			// same as the test above, except now we annotate
			// it in a different order.
			desc: "provide as with result annotation, in different order",
			provide: fx.Provide(
				fx.Annotate(func() *asStringer {
					return &asStringer{name: "stringer"}
				},
					fx.As(new(fmt.Stringer)),
					fx.ResultTags(`name:"goodStringer"`)),
			),
			invoke: func(i in) {
				assert.Equal(t, "stringer", i.S.String())
			},
		},
		{
			desc: "provide multiple constructors annotated As",
			provide: fx.Provide(
				fx.Annotate(func() *asStringer {
					return &asStringer{name: "stringer"}
				}, fx.As(new(fmt.Stringer))),
				fx.Annotate(func() *bytes.Buffer {
					buf := make([]byte, 1)
					return bytes.NewBuffer(buf)
				}, fx.As(new(io.Writer))),
			),
			invoke: func(s fmt.Stringer, w io.Writer) {
				assert.Equal(t, "stringer", s.String())
				_, err := w.Write([]byte{1})
				require.NoError(t, err)
			},
		},
		{
			desc: "provide the same provider as multiple types",
			provide: fx.Provide(
				fx.Annotate(newAsStringer, fx.As(new(fmt.Stringer))),
				fx.Annotate(newAsStringer, fx.As(new(myStringer))),
			),
			invoke: func(s fmt.Stringer, ms myStringer) {
				assert.Equal(t, "a good stringer", s.String())
				assert.Equal(t, "a good stringer", ms.String())
			},
		},
		{
			desc: "annotate fx.Supply",
			provide: fx.Supply(
				fx.Annotate(&asStringer{"foo"}, fx.As(new(fmt.Stringer))),
			),
			invoke: func(s fmt.Stringer) {
				assert.Equal(t, "foo", s.String())
			},
		},
		{
			desc: "annotate as many interfaces",
			provide: fx.Provide(
				fx.Annotate(func() *asStringer {
					return &asStringer{name: "stringer"}
				},
					fx.As(new(fmt.Stringer)),
					fx.As(new(myStringer))),
			),
			invoke: func(s fmt.Stringer, ms myStringer) {
				assert.Equal(t, "stringer", s.String())
				assert.Equal(t, "stringer", ms.String())
			},
		},
		{
			desc: "annotate as many interfaces with both-annotated return values",
			provide: fx.Provide(
				fx.Annotate(func() (*asStringer, *bytes.Buffer) {
					return &asStringer{name: "stringer"},
						bytes.NewBuffer(make([]byte, 1))
				},
					fx.As(new(fmt.Stringer), new(io.Reader)),
					fx.As(new(myStringer), new(io.Writer))),
			),
			invoke: func(s fmt.Stringer, ms myStringer, r io.Reader, w io.Writer) {
				assert.Equal(t, "stringer", s.String())
				assert.Equal(t, "stringer", ms.String())
				_, err := w.Write([]byte("."))
				assert.NoError(t, err)
				buf := make([]byte, 1)
				_, err = r.Read(buf)
				assert.NoError(t, err)
			},
		},
		{
			desc: "annotate as many interfaces with different numbers of annotations",
			provide: fx.Provide(
				fx.Annotate(func() (*asStringer, *bytes.Buffer) {
					return &asStringer{name: "stringer"},
						bytes.NewBuffer(make([]byte, 1))
				},
					// annotate both in here
					fx.As(new(fmt.Stringer), new(io.Writer)),
					// annotate just myStringer here
					fx.As(new(myStringer))),
			),
			invoke: func(s fmt.Stringer, ms myStringer, w io.Writer) {
				assert.Equal(t, "stringer", s.String())
				assert.Equal(t, "stringer", ms.String())
				_, err := w.Write([]byte("."))
				assert.NoError(t, err)
			},
		},
		{
			desc: "annotate many interfaces with varying annotation count and check original type",
			provide: fx.Provide(
				fx.Annotate(func() (*asStringer, *bytes.Buffer) {
					return &asStringer{name: "stringer"},
						bytes.NewBuffer(make([]byte, 1))
				},
					// annotate just myStringer here
					fx.As(new(myStringer)),
					// annotate both in here
					fx.As(new(fmt.Stringer), new(io.Writer))),
			),
			invoke: func(s fmt.Stringer, ms myStringer, buf *bytes.Buffer, w io.Writer) {
				assert.Equal(t, "stringer", s.String())
				assert.Equal(t, "stringer", ms.String())
				_, err := w.Write([]byte("."))
				assert.NoError(t, err)
				_, err = buf.Write([]byte("."))
				assert.NoError(t, err)
			},
		},
		{
			desc: "annotate fewer items than provided constructor",
			provide: fx.Provide(
				fx.Annotate(func() (*bytes.Buffer, *strings.Builder) {
					s := "Hello"
					return bytes.NewBuffer([]byte(s)), &strings.Builder{}
				},
					fx.As(new(io.Reader))),
			),
			invoke: func(r io.Reader) {
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			app := NewForTest(t,
				fx.WithLogger(func() fxevent.Logger {
					return fxtest.NewTestLogger(t)
				}),
				tt.provide,
				fx.Invoke(tt.invoke),
			)
			require.NoError(t, app.Err())
		})
	}
}

func TestAnnotatedAsFailures(t *testing.T) {
	t.Parallel()

	newAsStringer := func() *asStringer {
		return &asStringer{name: "stringer"}
	}

	tests := []struct {
		desc          string
		provide       fx.Option
		invoke        interface{}
		errorContains string
	}{
		{
			desc:          "provide when an illegal type As",
			provide:       fx.Provide(fx.Annotate(newAsStringer, fx.As(new(io.Writer)))),
			invoke:        func() {},
			errorContains: "does not implement",
		},
		{
			desc:    "don't provide original type using As",
			provide: fx.Provide(fx.Annotate(newAsStringer, fx.As(new(fmt.Stringer)))),
			invoke: func(as *asStringer) {
				fmt.Println(as.String())
			},
			errorContains: "missing type: *fx_test.asStringer",
		},
		{
			desc: "fail to provide with name annotation",
			provide: fx.Provide(fx.Annotate(func(n string) *asStringer {
				return &asStringer{name: n}
			}, fx.As(new(fmt.Stringer)), fx.ParamTags(`name:"n"`))),
			invoke: func(a fmt.Stringer) {
				fmt.Println(a)
			},
			errorContains: `missing type: string[name="n"]`,
		},
		{
			desc: "non-pointer argument to As",
			provide: fx.Provide(
				fx.Annotate(
					newAsStringer,
					fx.As("foo"),
				),
			),
			errorContains: "argument must be a pointer to an interface: got string",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			app := NewForTest(t,
				fx.WithLogger(func() fxevent.Logger {
					return fxtest.NewTestLogger(t)
				}),
				tt.provide,
				fx.Invoke(tt.invoke),
			)
			err := app.Err()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestAnnotatedWrongUsage(t *testing.T) {
	t.Parallel()

	type a struct {
		name string
	}
	type in struct {
		fx.In

		A *a `name:"foo"`
	}
	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("In Constructor", func(t *testing.T) {
		t.Parallel()

		var in in
		app := NewForTest(t,
			fx.WithLogger(func() fxevent.Logger {
				return fxtest.NewTestLogger(t)
			}),
			fx.Provide(
				func() fx.Annotated {
					return fx.Annotated{
						Name:   "foo",
						Target: newA,
					}
				},
			),
			fx.Populate(&in),
		)

		err := app.Err()
		require.Error(t, err)

		// Example:
		// fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor: fx.Provide received go.uber.org/fx_test.TestAnnotatedWrongUsage.func2.1() from:
		// go.uber.org/fx_test.TestAnnotatedWrongUsage.func2
		//         /.../fx/annotated_test.go:76
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor")
		assert.Contains(t, err.Error(), "fx.Provide received go.uber.org/fx_test.TestAnnotatedWrongUsage")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestAnnotatedWrongUsage")
		assert.Contains(t, err.Error(), "/annotated_test.go")
	})

	t.Run("Result Type", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.WithLogger(func() fxevent.Logger {
				return fxtest.NewTestLogger(t)
			}),
			fx.Provide(
				fx.Annotated{
					Name: "foo",
					Target: func() in {
						return in{A: &a{name: "foo"}}
					},
				},
			),
		)
		assert.Contains(t, app.Err().Error(), "embeds a dig.In", "expected error when result types were annotated")
	})

	t.Run("invalid group option", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(func() string { return "sad times" },
					fx.ResultTags(`group:"foo,soft"`)),
			),
		)
		assert.Contains(t, app.Err().Error(), "cannot use soft with result value groups", "expected error when invalid group option is provided")
	})
}

func TestAnnotatedString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give fx.Annotated
		want string
	}{
		{
			desc: "empty",
			give: fx.Annotated{},
			want: "fx.Annotated{}",
		},
		{
			desc: "name",
			give: fx.Annotated{Name: "foo"},
			want: `fx.Annotated{Name: "foo"}`,
		},
		{
			desc: "group",
			give: fx.Annotated{Group: "foo"},
			want: `fx.Annotated{Group: "foo"}`,
		},
		{
			desc: "name and group",
			give: fx.Annotated{Name: "foo", Group: "bar"},
			want: `fx.Annotated{Name: "foo", Group: "bar"}`,
		},
		{
			desc: "target",
			give: fx.Annotated{Target: func() {}},
			want: "fx.Annotated{Target: go.uber.org/fx_test.TestAnnotatedString.func1()}",
		},
		{
			desc: "name and target",
			give: fx.Annotated{Name: "foo", Target: func() {}},
			want: `fx.Annotated{Name: "foo", Target: go.uber.org/fx_test.TestAnnotatedString.func2()}`,
		},
		{
			desc: "group and target",
			give: fx.Annotated{Group: "foo", Target: func() {}},
			want: `fx.Annotated{Group: "foo", Target: go.uber.org/fx_test.TestAnnotatedString.func3()}`,
		},
		{
			desc: "name, group and target",
			give: fx.Annotated{Name: "foo", Group: "bar", Target: func() {}},
			want: `fx.Annotated{Name: "foo", Group: "bar", Target: go.uber.org/fx_test.TestAnnotatedString.func4()}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.give.String())
		})
	}
}

func TestAnnotate(t *testing.T) {
	t.Parallel()

	type a struct{}
	type b struct{ a *a }
	type c struct{ b *b }
	type sliceA struct{ sa []*a }
	newA := func() *a { return &a{} }
	newB := func(a *a) *b {
		return &b{a}
	}
	newC := func(b *b) *c {
		return &c{b}
	}
	newSliceA := func(sa ...*a) *sliceA {
		return &sliceA{sa}
	}
	newSliceAWithB := func(b *b, sa ...*a) *sliceA {
		total := append(sa, b.a)
		return &sliceA{total}
	}

	t.Run("Provide with optional", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(newB, fx.ParamTags(`name:"a" optional:"true"`)),
			),
			fx.Invoke(newC),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("Provide with many annotated params", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(newB, fx.ParamTags(`optional:"true"`)),
				fx.Annotate(func(a *a, b *b) interface{} { return nil },
					fx.ParamTags(`name:"a" optional:"true"`, `name:"b"`),
					fx.ResultTags(`name:"nil"`),
				),
			),
			fx.Invoke(newC),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("Invoke with optional", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Invoke(
				fx.Annotate(newB, fx.ParamTags(`optional:"true"`)),
			),
		)
		err := app.Err()
		require.NoError(t, err)
	})

	t.Run("Invoke with a missing dependency", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Invoke(
				fx.Annotate(newB, fx.ParamTags(`name:"a"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `missing dependencies`)
		assert.Contains(t, err.Error(), `missing type: *fx_test.a[name="a"]`)
	})

	t.Run("Provide with variadic function", func(t *testing.T) {
		t.Parallel()

		var got *sliceA
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{Group: "as", Target: newA},
				fx.Annotated{Group: "as", Target: newA},
				fx.Annotate(newSliceA,
					fx.ParamTags(`group:"as"`),
				),
			),
			fx.Populate(&got),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())

		assert.Len(t, got.sa, 2)
	})

	t.Run("Provide variadic function with no optional params", func(t *testing.T) {
		t.Parallel()

		var got struct {
			fx.In

			Result *sliceA `name:"as"`
		}
		app := fxtest.New(t,
			fx.Supply([]*a{{}, {}, {}}),
			fx.Provide(
				fx.Annotate(newSliceA,
					fx.ResultTags(`name:"as"`),
				),
			),
			fx.Populate(&got),
		)
		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
		assert.Len(t, got.Result.sa, 3)
	})

	t.Run("Provide variadic function named with no given params", func(t *testing.T) {
		t.Parallel()

		var got *sliceA
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(newSliceA, fx.ParamTags(`name:"a"`)),
			),
			fx.Populate(&got),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `missing dependencies`)
		assert.Contains(t, err.Error(), `missing type: []*fx_test.a[name="a"]`)
	})

	t.Run("Invoke function with soft group param", func(t *testing.T) {
		t.Parallel()
		newF := func(foos []int, bar string) {
			assert.ElementsMatch(t, []int{10}, foos)
		}
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() (int, string) { return 10, "hello" },
					fx.ResultTags(`group:"foos"`),
				),
				fx.Annotate(
					func() int {
						require.FailNow(t, "this function should not be called")
						return 20
					},
					fx.ResultTags(`group:"foos"`),
				),
			),
			fx.Invoke(
				fx.Annotate(newF, fx.ParamTags(`group:"foos,soft"`)),
			),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("Invoke variadic function with multiple params", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Supply(
				fx.Annotate(newB(newA()), fx.ResultTags(`name:"b"`)),
			),
			fx.Invoke(fx.Annotate(newSliceAWithB, fx.ParamTags(`name:"b"`))),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("Invoke non-optional variadic function with a missing dependency", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Invoke(
				fx.Annotate(newSliceA, fx.ParamTags(`optional:"false"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `missing dependencies`)
		assert.Contains(t, err.Error(), `missing type: []*fx_test.a`)
	})

	t.Run("Invoke with variadic function", func(t *testing.T) {
		t.Parallel()

		type T1 struct{ s string }

		app := fxtest.New(t,
			fx.Supply(
				fx.Annotate(T1{"foo"}, fx.ResultTags(`group:"t"`)),
				fx.Annotate(T1{"bar"}, fx.ResultTags(`group:"t"`)),
				fx.Annotate(T1{"baz"}, fx.ResultTags(`group:"t"`)),
			),
			fx.Invoke(fx.Annotate(func(got ...T1) {
				assert.ElementsMatch(t, []T1{{"foo"}, {"bar"}, {"baz"}}, got)
			}, fx.ParamTags(`group:"t"`))),
		)

		defer app.RequireStart().RequireStop()
		require.NoError(t, app.Err())
	})

	t.Run("provide with annotated results", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"firstA"`)),
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"secondA"`)),
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"thirdA"`)),
				fx.Annotate(func(a1 *a, a2 *a, a3 *a) *b {
					return &b{a1}
				}, fx.ParamTags(
					`name:"firstA"`,
					`name:"secondA"`,
					`name:"thirdA"`,
				)),
			),
			fx.Invoke(newC),
		)

		require.NoError(t, app.Err())
		defer app.RequireStart().RequireStop()
	})

	t.Run("provide with missing annotated results", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"firstA"`)),
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"secondA"`)),
				fx.Annotate(func() *a {
					return &a{}
				}, fx.ResultTags(`name:"fourthA"`)),
			),
			fx.Invoke(
				fx.Annotate(func(a1 *a, a2 *a, a3 *a) *b {
					return &b{a1}
				}, fx.ParamTags(
					`name:"firstA"`,
					`name:"secondA"`,
					`name:"thirdA"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `missing type: *fx_test.a[name="thirdA"]`)
	})

	t.Run("error in the middle of a function", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				//lint:ignore ST1008 we want to test error in the middle.
				fx.Annotate(func() (*a, error, *a) {
					return &a{}, nil, &a{}
				}, fx.ResultTags(`name:"firstA"`, ``, `name:"secondA"`)),
			),
			fx.Invoke(
				fx.Annotate(func(*a) {}, fx.ParamTags(`name:"firstA"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only the last result can be an error")
		assert.Contains(t, err.Error(), "returns error as result 1")
	})

	t.Run("provide with annotated results with error", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(func() (*a, *a, error) {
					return &a{}, &a{}, nil
				}, fx.ResultTags(`name:"firstA"`, `name:"secondA"`)),
				fx.Annotate(func() (*a, error) {
					return &a{}, nil
				}, fx.ResultTags(`name:"thirdA"`)),
			),
			fx.Invoke(fx.Annotate(func(a1 *a, a2 *a, a3 *a) *b {
				return &b{a2}
			}, fx.ParamTags(`name:"firstA"`, `name:"secondA"`, `name:"thirdA"`))))

		require.NoError(t, app.Err())
		defer app.RequireStart().RequireStop()
	})

	t.Run("provide an already provided function using Annotate", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(fx.Annotate(newA, fx.ResultTags(`name:"a"`))),
			fx.Provide(fx.Annotate(newA, fx.ResultTags(`name:"a"`))),
			fx.Invoke(
				fx.Annotate(newB, fx.ParamTags(`name:"a"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already provided")
		assert.Contains(t, err.Error(), "\"go.uber.org/fx_test\".TestAnnotate.func1")
	})

	t.Run("specify more ParamTags than Params", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			fx.Provide(
				// This should just leave newA as it is.
				fx.Annotate(newA, fx.ParamTags(`name:"something"`)),
			),
			fx.Invoke(newB),
		)

		err := app.Err()
		require.NoError(t, err)
		defer app.RequireStart().RequireStop()
	})

	t.Run("specify two ParamTags", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				// This should just leave newA as it is.
				fx.Annotate(
					newA,
					fx.ParamTags(`name:"something"`),
					fx.ParamTags(`name:"anotherThing"`),
				),
			),
			fx.Invoke(newB),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encountered error while applying annotation using fx.Annotate to go.uber.org/fx_test.TestAnnotate.func1(): cannot apply more than one line of ParamTags")
	})

	t.Run("specify two ResultTags", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				// This should just leave newA as it is.
				fx.Annotate(
					newA,
					fx.ResultTags(`name:"A"`),
					fx.ResultTags(`name:"AA"`),
				),
			),
			fx.Invoke(newB),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encountered error while applying annotation using fx.Annotate to go.uber.org/fx_test.TestAnnotate.func1(): cannot apply more than one line of ResultTags")
	})

	t.Run("annotate with a non-nil error", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(func() (*bytes.Buffer, error) {
					buf := make([]byte, 1)
					return bytes.NewBuffer(buf), errors.New("some error")
				}, fx.ResultTags(`name:"buf"`))),
			fx.Invoke(
				fx.Annotate(func(b *bytes.Buffer) {
					b.Write([]byte{1})
				}, fx.ParamTags(`name:"buf"`))),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "some error")
	})

	t.Run("annotate with a non-nil error and nil error", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(func() (*bytes.Buffer, error) {
					buf := make([]byte, 1)
					return bytes.NewBuffer(buf), errors.New("some error")
				}, fx.ResultTags(`name:"buf1"`)),
				fx.Annotate(func() (*bytes.Buffer, error) {
					buf := make([]byte, 1)
					return bytes.NewBuffer(buf), nil
				}, fx.ResultTags(`name:"buf2"`))),
			fx.Invoke(
				fx.Annotate(func(b1 *bytes.Buffer, b2 *bytes.Buffer) {
					b1.Write([]byte{1})
					b2.Write([]byte{1})
				}, fx.ParamTags(`name:"buf1"`, `name:"buf2"`))),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "some error")
	})

	t.Run("provide annotated non-function", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(42, fx.ResultTags(`name:"buf"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)

		// Example:
		// fx.Provide(fx.Annotate(42, fx.ResultTags(["name:\"buf\""])) from:
		// go.uber.org/fx_test.TestAnnotate.func17
		//     /Users/abg/dev/fx/annotated_test.go:697
		// testing.tRunner
		//     /usr/local/Cellar/go/1.17.2/libexec/src/testing/testing.go:1259
		// Failed: must provide constructor function, got 42 (int)

		assert.Contains(t, err.Error(), "fx.Provide(fx.Annotate(42")
		assert.Contains(t, err.Error(), "must provide constructor function, got 42 (int)")
	})

	t.Run("invoke annotated non-function", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			fx.Invoke(
				fx.Annotate(42, fx.ParamTags(`name:"buf"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide constructor function, got 42 (int)")
	})

	t.Run("annotate a fx.Out", func(t *testing.T) {
		t.Parallel()

		type A struct {
			s string

			fx.Out
		}

		f := func() A {
			return A{s: "hi"}
		}

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(f, fx.ResultTags(`name:"out"`)),
			),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fx.Out structs cannot be annotated")
	})

	t.Run("annotate a fx.In", func(t *testing.T) {
		t.Parallel()

		type A struct {
			S string
		}
		type B struct {
			fx.In
		}

		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(func(i A) string { return i.S }, fx.ParamTags(`optional:"true"`)),
				fx.Annotate(func(i B) string { return "ok" }, fx.ParamTags(`name:"problem"`)),
			),
		)
		err := app.Err()
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "invalid annotation function func(fx_test.A) string")
		assert.Contains(t, err.Error(), "invalid annotation function func(fx_test.B) string")
		assert.Contains(t, err.Error(), "fx.In structs cannot be annotated")
	})
}

func assertApp(
	t *testing.T,
	app interface {
		Start(context.Context) error
		Stop(context.Context) error
	},
	started *bool,
	stopped *bool,
	invoked *bool,
) {
	t.Helper()
	ctx := context.Background()
	assert.False(t, *started)
	require.NoError(t, app.Start(ctx))
	assert.True(t, *started)

	if invoked != nil {
		assert.True(t, *invoked)
	}

	if stopped != nil {
		assert.False(t, *stopped)
		require.NoError(t, app.Stop(ctx))
		assert.True(t, *stopped)
	}
}

func TestHookAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("with hook on invoke", func(t *testing.T) {
		t.Parallel()

		var started bool
		var invoked bool
		hook := fx.Annotate(
			func() {
				invoked = true
			},
			fx.OnStart(func(context.Context) error {
				started = true
				return nil
			}),
		)
		app := fxtest.New(t, fx.Invoke(hook))

		assertApp(t, app, &started, nil, &invoked)
	})

	t.Run("depend on result interface of target", func(t *testing.T) {
		type stub interface {
			String() string
		}

		var started bool

		hook := fx.Annotate(
			func() (stub, error) {
				b := []byte("expected")
				return bytes.NewBuffer(b), nil
			},
			fx.OnStart(func(_ context.Context, s stub) error {
				started = true
				require.Equal(t, "expected", s.String())
				return nil
			}),
		)

		app := fxtest.New(t,
			fx.Provide(hook),
			fx.Invoke(func(s stub) {
				require.Equal(t, "expected", s.String())
			}),
		)

		assertApp(t, app, &started, nil, nil)
	})

	t.Run("start and stop without dependencies", func(t *testing.T) {
		t.Parallel()

		type stub interface{}

		var (
			invoked bool
			started bool
			stopped bool
		)

		hook := fx.Annotate(
			func() (stub, error) { return nil, nil },
			fx.OnStart(func(context.Context) error {
				started = true
				return nil
			}),
			fx.OnStop(func(context.Context) error {
				stopped = true
				return nil
			}),
		)

		app := fxtest.New(t,
			fx.Provide(hook),
			fx.Invoke(func(s stub) {
				invoked = s == nil
			}),
		)

		assertApp(t, app, &started, &stopped, &invoked)
	})

	t.Run("with multiple extra dependency parameters", func(t *testing.T) {
		t.Parallel()

		type (
			A interface{}
			B interface{}
			C interface{}
		)

		var value int

		hook := fx.Annotate(
			func() (A, error) { return nil, nil },
			fx.OnStart(func(_ context.Context, b B, c C) error {
				b1, _ := b.(int)
				c1, _ := c.(int)
				value = b1 + c1
				return nil
			}),
		)

		app := fxtest.New(t,
			fx.Provide(hook),
			fx.Provide(func() B { return int(1) }),
			fx.Provide(func() C { return int(2) }),
			fx.Invoke(func(A) {}),
		)

		ctx := context.Background()
		assert.Zero(t, value)
		require.NoError(t, app.Start(ctx))
		defer func() {
			require.NoError(t, app.Stop(ctx))
		}()
		assert.Equal(t, 3, value)
	})

	t.Run("with Supply", func(t *testing.T) {
		t.Parallel()

		type A interface {
			WriteString(string) (int, error)
		}

		buf := bytes.NewBuffer(nil)
		var called bool

		ctor := fx.Provide(
			fx.Annotate(
				func() A {
					return buf
				},
				fx.OnStart(func(_ context.Context, a A, s fmt.Stringer) error {
					a.WriteString(s.String())
					return nil
				}),
			),
		)

		supply := fx.Supply(
			fx.Annotate(
				&asStringer{"supply"},
				fx.OnStart(func(context.Context) error {
					called = true
					return nil
				}),
				fx.As(new(fmt.Stringer)),
			))

		opts := fx.Options(
			ctor,
			supply,
			fx.Invoke(func(A) {}),
		)

		app := fxtest.New(t, opts)
		ctx := context.Background()
		require.False(t, called)
		err := app.Start(ctx)
		require.NoError(t, err)
		require.NoError(t, app.Stop(ctx))
		require.Equal(t, "supply", buf.String())
		require.True(t, called)
	})

	t.Run("with Decorate", func(t *testing.T) {
		t.Parallel()

		type A interface {
			WriteString(string) (int, error)
		}

		buf := bytes.NewBuffer(nil)
		ctor := fx.Provide(func() A { return buf })

		var called bool

		hook := fx.Annotate(
			func(in A) A {
				in.WriteString("decorated")
				return in
			},
			fx.OnStart(func(_ context.Context, a A) error {
				called = assert.Equal(t, "decorated", buf.String())
				return nil
			}),
		)

		decorated := fx.Decorate(hook)

		opts := fx.Options(
			ctor,
			decorated,
			fx.Invoke(func(A) {}),
		)

		app := fxtest.New(t, opts)
		ctx := context.Background()
		require.NoError(t, app.Start(ctx))
		require.NoError(t, app.Stop(ctx))
		require.True(t, called)
		require.Equal(t, "decorated", buf.String())
	})

	t.Run("with Supply and Decorate", func(t *testing.T) {
		t.Parallel()

		type A interface{}

		ch := make(chan string, 3)

		hook := fx.Annotate(
			func() A { return nil },
			fx.OnStart(func(_ context.Context, s fmt.Stringer) error {
				ch <- "constructor"
				require.Equal(t, "supply", s.String())
				return nil
			}),
		)

		ctor := fx.Provide(hook)

		hook = fx.Annotate(
			&asStringer{"supply"},
			fx.OnStart(func(_ context.Context) error {
				ch <- "supply"
				return nil
			}),
			fx.As(new(fmt.Stringer)),
		)

		supply := fx.Supply(hook)

		hook = fx.Annotate(
			func(in A) A { return in },
			fx.OnStart(func(_ context.Context) error {
				ch <- "decorated"
				return nil
			}),
		)

		decorated := fx.Decorate(hook)

		opts := fx.Options(
			ctor,
			supply,
			decorated,
			fx.Invoke(func(A) {}),
		)

		app := fxtest.New(t, opts)
		ctx := context.Background()
		err := app.Start(ctx)
		require.NoError(t, err)
		require.NoError(t, app.Stop(ctx))
		close(ch)

		require.Equal(t, "supply", <-ch)
		require.Equal(t, "constructor", <-ch)
		require.Equal(t, "decorated", <-ch)
	})
}

func TestHookAnnotationFailures(t *testing.T) {
	t.Parallel()
	validateApp := func(t *testing.T, opts ...fx.Option) error {
		return fx.ValidateApp(
			append(opts, fx.Logger(fxtest.NewTestPrinter(t)))...,
		)
	}

	type (
		A interface{}
		B interface{}
	)

	table := []struct {
		name        string
		annotation  interface{}
		useNew      bool
		errContains string
	}{
		{
			name:        "with unprovided dependency",
			errContains: "missing type: fx_test.B",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(func(context.Context, B) error {
					return nil
				}),
			),
		},
		{
			name:        "with hook that errors",
			errContains: "hook failed",
			useNew:      true,
			annotation: fx.Annotate(
				func() (A, error) { return nil, nil },
				fx.OnStart(func(context.Context) error {
					return errors.New("hook failed")
				}),
			),
		},
		{
			name:        "with multiple hooks of the same type",
			errContains: `cannot apply more than one "OnStart" hook annotation`,
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(func(context.Context) error { return nil }),
				fx.OnStart(func(context.Context) error { return nil }),
			),
		},
		{
			name:        "with hook that doesn't return an error",
			errContains: "must return only an error",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(func(context.Context) {}),
			),
		},
		{
			name:        "with constructor that errors",
			errContains: "hooks should not be installed",
			useNew:      true,
			annotation: fx.Annotate(
				func() (A, error) {
					return nil, errors.New("hooks should not be installed")
				},
				fx.OnStart(func(context.Context) error {
					require.FailNow(t, "hook should not be called")
					return nil
				}),
			),
		},
		{
			name:        "without a function target",
			errContains: "must provide function",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(&struct{}{}),
			),
		},
		{
			name:        "without context.Context as first parameter",
			errContains: "must be context.Context",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(func() {}),
			),
		},
		{
			name:        "with variactic hook",
			errContains: "must not accept variatic",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStart(func(context.Context, ...A) error {
					return nil
				}),
			),
		},
		{
			name:        "with nil hook target",
			errContains: "cannot use nil function",
			annotation: fx.Annotate(
				func() A { return nil },
				fx.OnStop(nil),
			),
		},
	}

	for _, tt := range table {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := fx.Options(
				fx.Provide(tt.annotation),
				fx.Invoke(func(A) {}),
			)

			if !tt.useNew {
				err := validateApp(t, opts)
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
				return
			}

			app := NewForTest(t, opts)
			err := app.Start(context.Background())
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errContains)
		})
	}
}
