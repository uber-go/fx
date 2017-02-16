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

type S struct{}

func noReturn() {}

func returnNonPointer() S {
	return S{}
}

func nonPointerParams(one, two string) *S {
	return &S{}
}

func TestInject(t *testing.T) {
	tts := []struct {
		name  string
		param interface{}
		err   error
	}{
		{"non function", struct{}{}, errParamType},
		{"wrong return count", noReturn, errReturnCount},
		{"non pointer return", returnNonPointer, errReturnKind},
		{"wrong parameters type", nonPointerParams, errArgKind},
		{"pointer inject", &struct{}{}, nil},
	}

	for _, tc := range tts {
		t.Run(tc.name, func(t *testing.T) {
			g := New()
			err := g.Inject(tc.param)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tts := []struct {
		name    string
		inject  func(g *graph) error
		resolve func(g *graph) error
		es      string
	}{
		{
			"non pointer resolve",
			func(g *graph) error { return g.Inject(NewParent1) },
			func(g *graph) error { return g.Resolve(S{}) },
			"can not resolve non-pointer object",
		},
		{
			"missing dependency",
			func(g *graph) error { return nil },
			func(g *graph) error {
				var p1 *Parent1
				return g.Resolve(p1)
			},
			"type *dig.Parent1 is not registered",
		},
	}

	for _, tc := range tts {
		t.Run(tc.name, func(t *testing.T) {
			g := testGraph()

			err := tc.inject(g)
			require.NoError(t, err, "Inject part of the test cant have errors")

			err = tc.resolve(g)
			if err != nil {
				require.Contains(t, err.Error(), tc.es, "Unexpected error message")
			} else {
				require.NoError(t, err, "Did not expect a resolve error")
			}
		})
	}
}

func TestObjectInject(t *testing.T) {
	g := New()

	// inject a fake struct type into the graph
	type Fake struct {
		Name string
	}
	err := g.Inject(&Fake{Name: "I am a fake injected thing"})
	require.NoError(t, err)

	// get one pointer resolved
	var f1 *Fake
	err = g.Resolve(&f1)
	require.NoError(t, err)

	// get second pointer resolved
	var f2 *Fake
	err = g.Resolve(&f2)
	require.NoError(t, err)

	// make sure they are the same
	require.Equal(t, f1, f2)
	require.Equal(t, *f1, *f2)
}

func TestConstructorInject(t *testing.T) {
	g := testGraph()

	err := g.Inject(NewGrandchild1)
	require.NoError(t, err)

	var gc1 *Grandchild1
	err = g.Resolve(&gc1)

	require.NoError(t, err, "No error expected during Resolve")
	require.NotNil(t, gc1, "Child1 must have been injected")
}

func TestBasicResolve(t *testing.T) {
	g := testGraph()

	err := g.InjectAll(
		NewParent1,
		NewChild1,
		NewGrandchild1,
	)
	require.NoError(t, err)

	var p1 *Parent1
	err = g.Resolve(&p1)

	require.NoError(t, err, "No error expected during Resolve")
	require.NotNil(t, p1.c1, "Child1 must have been injected")
	require.NotNil(t, p1.c1.gc1, "Grandchild1 must have been injected")
}

func testGraph() *graph {
	return &graph{
		nodes: make(map[interface{}]object),
	}
}
