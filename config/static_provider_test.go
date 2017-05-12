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
	"github.com/stretchr/testify/require"
)

func TestStaticProvider_Name(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(nil)
	assert.Equal(t, "static", p.Name())
}

func TestNewStaticProvider_NilData(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(nil)

	val := p.Get("something")
	assert.False(t, val.HasValue())
}

func TestStaticProvider_WithData(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"hello": "world",
	}
	p := NewStaticProvider(data)

	val := p.Get("hello")
	assert.True(t, val.HasValue())
	assert.False(t, val.IsDefault())
	assert.Equal(t, "world", val.AsString())
}

func TestStaticProvider_WithGet(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"hello": map[string]int{"world": 42},
	}
	p := NewStaticProvider(data)

	val := p.Get("hello")
	assert.True(t, val.HasValue())

	sub := p.Get("hello")
	val = sub.Get("world")
	assert.True(t, val.HasValue())
	assert.Equal(t, 42, val.AsInt())
}

func TestStaticProvider_Callbacks(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}

func TestStaticProviderFmtPrintOnValueNoPanic(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(nil)
	val := p.Get("something")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", val), "")
	}
	assert.NotPanics(t, f)
}

func TestNilStaticProviderSetDefaultTagValue(t *testing.T) {
	t.Parallel()
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
		ID6 [6]Inner        `yaml:"id6"`
		ID7 [7]*Inner       `yaml:"id7"`
	}{}

	p := NewStaticProvider(nil)
	require.NoError(t, p.Get("hello").Populate(&data))

	assert.Equal(t, 10, data.ID0)
	assert.Equal(t, "string", data.ID1)
	assert.True(t, data.ID2.Set)
	assert.Nil(t, data.ID3)
	assert.Nil(t, data.ID4)
	assert.Nil(t, data.ID5)
	assert.True(t, data.ID6[0].Set)
	assert.Nil(t, data.ID7[0])
}

func TestPopulateForSimpleMap(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(map[string]int{"one": 1, "b": -1})

	var m map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Equal(t, 1, m["one"])
}

func TestPopulateForNestedMap(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(map[string]interface{}{
		"top":    map[string]int{"one": 1, "": -1},
		"bottom": "value"})

	var m map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Equal(t, 2, len(m["top"].(map[interface{}]interface{})))
	assert.Equal(t, 1, m["top"].(map[interface{}]interface{})["one"])
	assert.Equal(t, "value", m["bottom"])
}

func TestPopulateForSimpleSlice(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider([]string{"Eeny", "meeny", "miny", "moe"})

	var s []string
	require.NoError(t, p.Get(Root).Populate(&s))
	assert.Equal(t, []string{"Eeny", "meeny", "miny", "moe"}, s)

	var str string
	require.NoError(t, p.Get("1").Populate(&str))
	assert.Equal(t, "meeny", str)
	assert.Equal(t, "miny", p.Get("2").String())
}

func TestPopulateForNestedSlices(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider([][]string{{}, {"Catch", "a", "tiger", "by", "the", "toe"}, nil, {""}})

	var s [][]string
	require.NoError(t, p.Get(Root).Populate(&s))
	require.Equal(t, 4, len(s))
	assert.Equal(t, [][]string{nil, {"Catch", "a", "tiger", "by", "the", "toe"}, nil, {""}}, s)
	assert.Equal(t, "Catch", p.Get("1.0").String())
}

func TestPopulateForBuiltins(t *testing.T) {
	t.Parallel()
	t.Run("int", func(t *testing.T) {
		p := NewStaticProvider(1)
		var i int
		require.NoError(t, p.Get(Root).Populate(&i))
		assert.Equal(t, 1, i)
		assert.Equal(t, 1, p.Get(Root).AsInt())
	})
	t.Run("float", func(t *testing.T) {
		p := NewStaticProvider(1.23)
		var f float64
		require.NoError(t, p.Get(Root).Populate(&f))
		assert.Equal(t, 1.23, f)
		assert.Equal(t, 1.23, p.Get(Root).AsFloat())
	})
	t.Run("string", func(t *testing.T) {
		p := NewStaticProvider("pie")
		var s string
		require.NoError(t, p.Get(Root).Populate(&s))
		assert.Equal(t, "pie", s)
		assert.Equal(t, "pie", p.Get(Root).String())
	})
	t.Run("bool", func(t *testing.T) {
		p := NewStaticProvider(true)
		var b bool
		require.NoError(t, p.Get(Root).Populate(&b))
		assert.True(t, b)
		assert.True(t, p.Get(Root).AsBool())
	})
}

func TestPopulateForNestedMaps(t *testing.T) {
	t.Parallel()
	p := NewStaticProvider(map[string]map[string]string{
		"a": {"one": "1", "": ""}})

	var m map[string]map[string]string
	err := p.Get("a").Populate(&m)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `empty map key is ambigious`)
	assert.Contains(t, err.Error(), `a.`)
}

func TestPopulateNonPointerType(t *testing.T) {
	t.Parallel()

	p := NewStaticProvider(42)
	x := 13
	err := p.Get(Root).Populate(x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't populate non pointer type")
}

func TestStaticProviderWithExpand(t *testing.T) {
	t.Parallel()

	p := NewStaticProviderWithExpand(map[string]interface{}{
		"slice": []interface{}{"one", "${iTwo:2}"},
		"value": `${iValue:""}`,
		"map": map[string]interface{}{
			"drink?": "${iMap:tea?}",
			"tea?":   "with cream",
		},
	}, func(key string) (string, bool) {
		switch key {
		case "iValue":
			return "null", true
		case "iTwo":
			return "3", true
		case "iMap":
			return "rum please!", true
		}

		return "", false
	})

	assert.Equal(t, "one", p.Get("slice.0").AsString())
	assert.Equal(t, "3", p.Get("slice.1").AsString())
	assert.Equal(t, "null", p.Get("value").Value())

	assert.Equal(t, "rum please!", p.Get("map.drink?").AsString())
	assert.Equal(t, "with cream", p.Get("map.tea?").AsString())
}
