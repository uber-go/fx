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

		// Exmaple:
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
}
