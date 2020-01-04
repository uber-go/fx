// Copyright (c) 2020 Uber Technologies, Inc.
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

package fxlog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpy(t *testing.T) {
	var s Spy

	t.Run("empty spy", func(t *testing.T) {
		assert.Empty(t, s.Messages(), "messages must be empty")
		assert.Empty(t, s.String(), "string must be empty")
	})

	s.Printf("foo bar")
	t.Run("unformatted message", func(t *testing.T) {
		assert.Equal(t, []string{"foo bar"}, s.Messages(), "messages must match")
		assert.Equal(t, "foo bar\n", s.String(), "string must match")
	})

	s.Printf("something went wrong: %v", errors.New("great sadness"))
	t.Run("formatted message", func(t *testing.T) {
		assert.Equal(t, []string{
			"foo bar",
			"something went wrong: great sadness",
		}, s.Messages())
		assert.Equal(t, "foo bar\nsomething went wrong: great sadness\n", s.String())
	})

	s.Reset()
	t.Run("reset", func(t *testing.T) {
		assert.Empty(t, s.Messages(), "messages must be empty")
		assert.Empty(t, s.String(), "string must be empty")
	})

	s.Printf("baz qux")
	t.Run("use after reset", func(t *testing.T) {
		assert.Equal(t, []string{"baz qux"}, s.Messages(), "messages must match")
		assert.Equal(t, "baz qux\n", s.String(), "string must match")
	})
}
