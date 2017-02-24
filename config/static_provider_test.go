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

func TestNilStaticProviderSetDefaultTagValue(t *testing.T) {
	type Inner struct {
		Set bool `yaml:"set" default:"true"`
	}
	data := struct {
		ID0 int             `yaml:"id0" default:"10"`
		ID1 string          `yaml:"id1" default:"string"`
		ID2 Inner           `yaml:"id2"`
		ID3 []Inner         `yaml:"id3"`
		ID4 map[Inner]Inner `yaml:"id4"`
		ID5 *Inner          `yaml:"id5"`
		//ID6 [6]Inner        `yaml:"id6"`
		ID7 [7]*Inner `yaml:"id7"`
	}{}

	p := NewStaticProvider(nil)
	p.Get("hello").PopulateStruct(&data)

	assert.Equal(t, 10, data.ID0)
	assert.Equal(t, "string", data.ID1)
	assert.True(t, data.ID2.Set)
	assert.Nil(t, data.ID3)
	assert.Nil(t, data.ID4)
	assert.Nil(t, data.ID5)
	// TODO (yutong) uncomment following assert after DRI-12.
	// assert.True(t, data.ID6[0].Set)
	assert.Nil(t, data.ID7[0])
}
