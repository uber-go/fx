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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupProvider_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "lookup", NewLookupProvider(nil).Name())
}

func TestLookupProvider_Get(t *testing.T) {
	t.Parallel()

	f := func(key string) (interface{}, bool) {
		if key == "Life was like a box of chocolates" {
			return "you never know what you're gonna get", true
		}

		return nil, false
	}

	p := NewLookupProvider(f)
	assert.Equal(t, "you never know what you're gonna get",
		p.Get("Life was like a box of chocolates").AsString())

	assert.Equal(t, false, p.Get("Forrest Gump").HasValue())
}
