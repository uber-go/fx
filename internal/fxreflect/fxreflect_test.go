// Copyright (c) 2017 Uber Technologies, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReturnTypes(t *testing.T) {
	t.Run("Primitive", func(t *testing.T) {
		fn := func() (int, string) {
			return 0, ""
		}
		assert.Equal(t, []string{"int", "string"}, ReturnTypes(fn))
	})
	t.Run("Pointer", func(t *testing.T) {
		type s struct{}
		fn := func() *s {
			return &s{}
		}
		assert.Equal(t, []string{"*fxreflect.s"}, ReturnTypes(fn))
	})
	t.Run("Interface", func(t *testing.T) {
		fn := func() hollerer {
			return impl{}
		}
		assert.Equal(t, []string{"fxreflect.hollerer"}, ReturnTypes(fn))
	})
	t.Run("SkipsErr", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("err")
		}
		assert.Equal(t, []string{"string"}, ReturnTypes(fn))
	})
}

type hollerer interface {
	Holler()
}

type impl struct{}

func (impl) Holler() {}

func TestCaller(t *testing.T) {
	assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestCaller", Caller())
}

func someFunc() {}

func TestFuncName(t *testing.T) {
	assert.Equal(t, "go.uber.org/fx/internal/fxreflect.someFunc()", FuncName(someFunc))
	assert.Equal(t, "n/a", FuncName(struct{}{}))
}
