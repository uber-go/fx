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
	"math"
	"reflect"
	"testing"

	"github.com/cheekybits/genny/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type overflowType generic.Type

//go:generate genny -in=$GOFILE -out=gen-$GOFILE gen "overflowType=int,uint,int8,uint8,int16,uint16,int32,uint32,int64,uint64,uintptr,float32"

func TestNumbersOverflowParsing_overflowType(t *testing.T) {
	t.Parallel()

	var x overflowType
	// A trick to not run test in generic case
	if reflect.ValueOf(x).Kind() == reflect.Invalid {
		t.Skip()
	}

	p := newValueProvider(math.MaxFloat64)
	err := p.Get("").PopulateStruct(&x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(`can't convert %q`, fmt.Sprint(math.MaxFloat64)))
	assert.Contains(t, err.Error(), "overflowType")
}
