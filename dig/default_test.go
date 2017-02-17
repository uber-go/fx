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

package dig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Type1 struct {
	t int
}

type Type2 struct {
	s string
}

type Type3 struct {
	f float32
}

func TestDefaultGraph(t *testing.T) {
	defer Reset()

	t1 := &Type1{t: 42}
	require.NoError(t, Register(t1))

	t2 := &Type2{s: "42"}
	t3 := &Type3{f: 4.2}
	require.NoError(t, RegisterAll(t2, t3))

	var t1g *Type1
	require.NoError(t, Resolve(&t1g))
	require.True(t, t1g == t1)

	var t2g *Type2
	var t3g *Type3
	require.NoError(t, ResolveAll(&t2g, &t3g))
	require.True(t, t2g == t2)
	require.True(t, t3g == t3)
}
