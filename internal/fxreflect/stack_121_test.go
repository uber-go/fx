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

//go:build go1.21
// +build go1.21

package fxreflect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepStack(t *testing.T) {
	t.Run("nest", func(t *testing.T) {
		// Introduce a few frames.
		frames := func() []Frame {
			return func() []Frame {
				return CallerStack(0, 0)
			}()
		}()

		require.True(t, len(frames) > 3, "expected at least three frames")
		for i, name := range []string{"func1.TestDeepStack.func1.1.func2", "func1.1", "func1"} {
			f := frames[i]
			assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestDeepStack."+name, f.Function)
			assert.Contains(t, f.File, "internal/fxreflect/stack_121_test.go")
			assert.NotZero(t, f.Line)
		}
	})
}
