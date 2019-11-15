// Copyright (c) 2019 Uber Technologies, Inc.
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

package fxreflect

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStack(t *testing.T) {
	// NOTE:
	// We don't assert the length of the stack because we cannot make
	// guarantees about how many frames the test runner is allowed to
	// introduce.

	t.Run("default", func(t *testing.T) {
		frames := CallerStack(0, 0)
		require.NotEmpty(t, frames)
		f := frames[0]
		assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestStack.func1", f.Function)
		assert.Contains(t, f.File, "internal/fxreflect/stack_test.go")
		assert.NotZero(t, f.Line)
	})

	t.Run("default/deeper", func(t *testing.T) {
		// Introduce a few frames.
		frames := func() []Frame {
			return func() []Frame {
				return CallerStack(0, 0)
			}()
		}()

		require.True(t, len(frames) > 3, "expected at least three frames")
		for i, name := range []string{"func2.1.1", "func2.1", "func2"} {
			f := frames[i]
			assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestStack."+name, f.Function)
			assert.Contains(t, f.File, "internal/fxreflect/stack_test.go")
			assert.NotZero(t, f.Line)
		}
	})

	t.Run("skip", func(t *testing.T) {
		// Introduce a few frames and skip 2.
		frames := func() []Frame {
			return func() []Frame {
				return CallerStack(2, 0)
			}()
		}()

		require.NotEmpty(t, frames)
		f := frames[0]
		assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestStack.func3", f.Function)
		assert.Contains(t, f.File, "internal/fxreflect/stack_test.go")
		assert.NotZero(t, f.Line)
	})
}

func TestStackCallerName(t *testing.T) {
	tests := []struct {
		desc string
		give Stack
		want string
	}{
		{desc: "empty", want: "n/a"},
		{
			desc: "skip Fx components",
			give: Stack{
				{
					Function: "go.uber.org/fx.Foo()",
					File:     "go.uber.org/fx/foo.go",
				},
				{
					Function: "foo/bar.Baz()",
					File:     "foo/bar/baz.go",
				},
			},
			want: "foo/bar.Baz()",
		},
		{
			desc: "skip Fx in wrong directory",
			give: Stack{
				{
					Function: "go.uber.org/fx.Foo()",
					File:     "fx/foo.go",
				},
				{
					Function: "foo/bar.Baz()",
					File:     "foo/bar/baz.go",
				},
			},
			want: "foo/bar.Baz()",
		},
		{
			desc: "skip Fx subpackage",
			give: Stack{
				{
					Function: "go.uber.org/fx/internal/fxreflect.Foo()",
					File:     "fx/internal/fxreflect/foo.go",
				},
				{
					Function: "foo/bar.Baz()",
					File:     "foo/bar/baz.go",
				},
			},
			want: "foo/bar.Baz()",
		},
		{
			desc: "don't skip Fx tests",
			give: Stack{
				{
					Function: "some/thing.Foo()",
					File:     "go.uber.org/fx/foo_test.go",
				},
			},
			want: "some/thing.Foo()",
		},
		{
			desc: "don't skip fx prefix",
			give: Stack{
				{
					Function: "go.uber.org/fxfoo.Bar()",
					File:     "go.uber.org/fxfoo/bar.go",
				},
			},
			want: "go.uber.org/fxfoo.Bar()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.give.CallerName())
		})
	}
}

func TestFrameString(t *testing.T) {
	tests := []struct {
		desc string
		give Frame
		want string
	}{
		{
			desc: "zero",
			give: Frame{},
			want: "unknown",
		},
		{
			desc: "file and line",
			give: Frame{File: "foo.go", Line: 42},
			want: "(foo.go:42)",
		},
		{
			desc: "file only",
			give: Frame{File: "foo.go"},
			want: "(foo.go)",
		},
		{
			desc: "function only",
			give: Frame{Function: "foo"},
			want: "foo",
		},
		{
			desc: "function and file",
			give: Frame{Function: "foo", File: "bar.go"},
			want: "foo (bar.go)",
		},
		{
			desc: "function and line",
			give: Frame{Function: "foo", Line: 42},
			want: "foo", // line without file is meaningless
		},
		{
			desc: "function, file, and line",
			give: Frame{Function: "foo", File: "bar.go", Line: 42},
			want: "foo (bar.go:42)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.give.String())
		})
	}
}

func TestStackFormat(t *testing.T) {
	stack := Stack{
		{
			Function: "path/to/module.SomeFunction()",
			File:     "path/to/file.go",
			Line:     42,
		},
		{
			Function: "path/to/another/module.AnotherFunction()",
			File:     "path/to/another/file.go",
			Line:     12,
		},
	}

	t.Run("single line", func(t *testing.T) {
		assert.Equal(t,
			"path/to/module.SomeFunction() (path/to/file.go:42); "+
				"path/to/another/module.AnotherFunction() (path/to/another/file.go:12)",
			fmt.Sprintf("%v", stack))
	})

	t.Run("multi line", func(t *testing.T) {
		assert.Equal(t, `path/to/module.SomeFunction()
	path/to/file.go:42
path/to/another/module.AnotherFunction()
	path/to/another/file.go:12
`, fmt.Sprintf("%+v", stack))
	})

}
