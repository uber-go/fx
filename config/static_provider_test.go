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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticProvider_Name(t *testing.T) {
	p := NewStaticProvider(nil)
	assert.Equal(t, "static", p.Name())
}

func TestNewStaticProvider_NilData(t *testing.T) {
	p := NewStaticProvider(nil)

	val := p.Get("something")
	assert.False(t, val.HasValue())
}

func TestStaticProvider_WithData(t *testing.T) {
	data := map[string]interface{}{
		"hello": "world",
	}
	p := NewStaticProvider(data)

	val := p.Get("hello")
	assert.True(t, val.HasValue())
	assert.False(t, val.IsDefault())
	assert.Equal(t, "world", val.AsString())
}

func TestStaticProvider_WithScope(t *testing.T) {
	data := map[string]interface{}{
		"hello.world": 42,
	}
	p := NewStaticProvider(data)

	val := p.Get("hello")
	assert.False(t, val.HasValue())

	sub := p.Scope("hello")
	val = sub.Get("world")
	assert.True(t, val.HasValue())
	assert.Equal(t, 42, val.AsInt())
}

func TestStaticProvider_Callbacks(t *testing.T) {
	p := NewStaticProvider(nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}

func TestStaticProviderFmtPrintOnValueNoPanic(t *testing.T) {
	p := NewStaticProvider(nil)
	val := p.Get("something")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", val), "")
	}
	assert.NotPanics(t, f)
}
