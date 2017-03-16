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
	"math"
)

func TestNumericConversion(t *testing.T) {
	t.Parallel()

	providers := map[string]Provider{
		"int":     newValueProvider(int8(1)),
		"uint":    newValueProvider(uint(1)),
		"int8":    newValueProvider(int8(1)),
		"uint8":   newValueProvider(uint8(1)),
		"int16":   newValueProvider(int16(1)),
		"uint16":  newValueProvider(uint16(1)),
		"int32":   newValueProvider(int32(1)),
		"uint32":  newValueProvider(uint32(1)),
		"int64":   newValueProvider(int64(1)),
		"uint64":  newValueProvider(uint64(1)),
		"float32": newValueProvider(float32(1)),
		"float64": newValueProvider(float64(1)),
		"uintptr": newValueProvider(uintptr(1)),
	}

	conversions := map[string]func(p Provider, t *testing.T){
		"int": func(p Provider, t *testing.T) {
			var x int
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, int(1), x)
		},
		"uint": func(p Provider, t *testing.T) {
			var x uint
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uint(1), x)
		},
		"int8": func(p Provider, t *testing.T) {
			var x int8
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, int8(1), x)
		},
		"uint8": func(p Provider, t *testing.T) {
			var x uint8
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uint8(1), x)
		},
		"int16": func(p Provider, t *testing.T) {
			var x int16
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, int16(1), x)
		},
		"uint16": func(p Provider, t *testing.T) {
			var x uint16
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uint16(1), x)
		},
		"int32": func(p Provider, t *testing.T) {
			var x int32
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, int32(1), x)
		},
		"uint32": func(p Provider, t *testing.T) {
			var x uint32
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uint32(1), x)
		},
		"int64": func(p Provider, t *testing.T) {
			var x int64
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, int64(1), x)
		},
		"uint64": func(p Provider, t *testing.T) {
			var x uint64
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uint64(1), x)
		},
		"float32": func(p Provider, t *testing.T) {
			var x float32
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, float32(1), x)
		},
		"float64": func(p Provider, t *testing.T) {
			var x float64
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, float64(1), x)
		},
		"uintptr": func(p Provider, t *testing.T) {
			var x uintptr
			require.NoError(t, p.Get(Root).PopulateStruct(&x))
			assert.Equal(t, uintptr(1), x)
		},
	}

	for from, provider := range providers {
		for to, test := range conversions {
			assert.True(t, t.Run(
				fmt.Sprintf("From %q to %q", from, to),
				func(t *testing.T) {
					test(provider, t)
				}),
			)
		}
	}
}

func TestNumericOverflows(t *testing.T) {
	p := newValueProvider(math.MaxFloat64)

	conversions := map[string]func(p Provider) error{
		"int": func(p Provider) error {
			var x int
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint": func(p Provider) error {
			var x uint
			return p.Get(Root).PopulateStruct(&x)
		},
		"int8": func(p Provider) error {
			var x int8
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint8": func(p Provider) error {
			var x uint8
			return p.Get(Root).PopulateStruct(&x)
		},
		"int16": func(p Provider) error {
			var x int16
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint16": func(p Provider) error {
			var x uint16
			return p.Get(Root).PopulateStruct(&x)
		},
		"int32": func(p Provider) error {
			var x int32
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint32": func(p Provider) error {
			var x uint32
			return p.Get(Root).PopulateStruct(&x)
		},
		"int64": func(p Provider) error {
			var x int64
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint64": func(p Provider) error {
			var x uint64
			return p.Get(Root).PopulateStruct(&x)
		},
		"float32": func(p Provider) error {
			var x float32
			return p.Get(Root).PopulateStruct(&x)
		},
		"uintptr": func(p Provider) error {
			var x uintptr
			return p.Get(Root).PopulateStruct(&x)
		},
	}

	for to, f := range conversions {
		assert.True(t, t.Run(fmt.Sprintf("%q overflow", to), func(t *testing.T){
			err := f(p)
			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf(`can't convert %q`, fmt.Sprint(math.MaxFloat64)))
			assert.Contains(t, err.Error(), to)
		}))
	}
}


func TestUnsignedNumericDecodingNegatives(t *testing.T) {
	p := newValueProvider(-1)

	conversions := map[string]func(p Provider) error{
		"uint": func(p Provider) error {
			var x uint
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint8": func(p Provider) error {
			var x uint8
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint16": func(p Provider) error {
			var x uint16
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint32": func(p Provider) error {
			var x uint32
			return p.Get(Root).PopulateStruct(&x)
		},
		"uint64": func(p Provider) error {
			var x uint64
			return p.Get(Root).PopulateStruct(&x)
		},
		"uintptr": func(p Provider) error {
			var x uintptr
			return p.Get(Root).PopulateStruct(&x)
		},
	}

	for to, f := range conversions {
		assert.True(t, t.Run(fmt.Sprintf("%q convert negative", to), func(t *testing.T){
			err := f(p)
			require.Error(t, err)
			assert.Contains(t, fmt.Sprintf("can't convert \"-1\" to unsigned integer type %q", to), err.Error())
		}))
	}
}
