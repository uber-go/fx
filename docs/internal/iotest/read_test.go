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

package iotest

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/docs/internal/test"
)

func TestReadAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		got := ReadAll(t, strings.NewReader("hello"))
		assert.Equal(t, "hello", got)
	})

	t.Run("failure", func(t *testing.T) {
		newT := fakeT{T: t}
		ReadAll(&newT, iotest.ErrReader(errors.New("great sadness")))
		assert.True(t, newT.failed, "test must have failed")
	})
}

type fakeT struct {
	test.T

	failed bool
}

func (t *fakeT) Errorf(msg string, args ...any) {
	t.failed = true
}

func (t *fakeT) FailNow() {
	t.failed = true
}
