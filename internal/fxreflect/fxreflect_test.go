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
	"errors"
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
)

type hollerer interface {
	Holler()
}

type impl struct{}

func (impl) Holler() {}

type result struct {
	dig.Out // referencing fx introduces import cycles

	mu     sync.Mutex // unexported
	Logger *log.Logger
}

func TestReturnTypes(t *testing.T) {
	t.Run("non-function", func(t *testing.T) {
		assert.Empty(t, ReturnTypes(42))
	})
	t.Run("primitive", func(t *testing.T) {
		fn := func() (int, string) {
			return 0, ""
		}
		assert.Equal(t, []string{"int", "string"}, ReturnTypes(fn))
	})
	t.Run("pointer", func(t *testing.T) {
		type s struct{}
		fn := func() *s {
			return &s{}
		}
		assert.Equal(t, []string{"*fxreflect.s"}, ReturnTypes(fn))
	})
	t.Run("interface", func(t *testing.T) {
		fn := func() hollerer {
			return impl{}
		}
		assert.Equal(t, []string{"fxreflect.hollerer"}, ReturnTypes(fn))
	})
	t.Run("result struct", func(t *testing.T) {
		fn := func() result {
			return result{}
		}
		assert.Equal(t, []string{"*log.Logger"}, ReturnTypes(fn))
	})
	t.Run("skips errors", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("err")
		}
		assert.Equal(t, []string{"string"}, ReturnTypes(fn))
	})
}

func TestCaller(t *testing.T) {
	assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestCaller", Caller())
}

func someFunc() {}

func TestFuncName(t *testing.T) {
	tests := []struct {
		desc string
		give interface{}
		want string
	}{
		{
			desc: "function",
			give: someFunc,
			want: "go.uber.org/fx/internal/fxreflect.someFunc()",
		},
		{
			desc: "not a function",
			give: 42,
			want: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.want, FuncName(tt.give))
		})
	}
}

func TestSanitizeFuncNames(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"url encoding",
			"go.uber.org/fx/sample%2egit/someFunc",
			"go.uber.org/fx/sample.git/someFunc",
		},
		{
			"vendor removal",
			"go.uber.org/fx/vendor/github.com/some/lib.SomeFunc",
			"vendor/github.com/some/lib.SomeFunc",
		},
		{
			"package happens to be named vendor is untouched",
			"go.uber.org/fx/foovendor/someFunc",
			"go.uber.org/fx/foovendor/someFunc",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, sanitize(c.input))
		})
	}
}
