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

func TestRegister(t *testing.T) {
	t.Parallel()
	tts := []struct {
		name  string
		param interface{}
		err   error
	}{
		{"non function", struct{}{}, errParamType},
		{"wrong return count", noReturn, errReturnCount},
		{"non pointer return", returnNonPointer, errReturnKind},
		{"wrong parameters type", nonPointerParams, errArgKind},
		{"pointer register", &struct{}{}, nil},
	}

	for _, tc := range tts {
		t.Run(tc.name, func(t *testing.T) {
			g := New()
			err := g.Register(tc.param)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	tts := []struct {
		name     string
		register func(g *Graph) error
		resolve  func(g *Graph) error
		es       string
	}{
		{
			"non pointer resolve",
			func(g *Graph) error { return g.Register(NewParent1) },
			func(g *Graph) error { return g.Resolve(S{}) },
			"can not resolve non-pointer object",
		},
		{
			"missing dependency",
			func(g *Graph) error { return nil },
			func(g *Graph) error {
				var p1 *Parent1
				return g.Resolve(p1)
			},
			"type *dig.Parent1 is not registered",
		},
	}

	for _, tc := range tts {
		t.Run(tc.name, func(t *testing.T) {
			g := New()

			err := tc.register(g)
			require.NoError(t, err, "Register part of the test cant have errors")

			err = tc.resolve(g)
			if err != nil {
				require.Contains(t, err.Error(), tc.es, "Unexpected error message")
			} else {
				require.NoError(t, err, "Did not expect a resolve error")
			}
		})
	}
}

func TestObjectRegister(t *testing.T) {
	t.Parallel()
	g := New()

	// register a fake struct type into the graph
	type Fake struct {
		Name string
	}
	err := g.Register(&Fake{Name: "I am a fake registered thing"})
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

func TestBasicRegisterResolve(t *testing.T) {
	t.Parallel()
	g := New()

	err := g.Register(NewGrandchild1)
	require.NoError(t, err)

	var first *Grandchild1
	require.NoError(t, g.Resolve(&first), "No error expected during first Resolve")

	var second *Grandchild1
	require.NoError(t, g.Resolve(&second), "No error expected during second Resolve")

	require.NotNil(t, first, "Child1 must have been registered")
	require.NotNil(t, second, "Child1 must have been registered")
	require.True(t, first == second, "Must point to the same object")
}

func TestRegisterAll(t *testing.T) {
	t.Parallel()
	g := New()

	err := g.RegisterAll(
		NewParent1,
		NewChild1,
		NewGrandchild1,
	)
	require.NoError(t, err)

	var p1 *Parent1
	err = g.Resolve(&p1)

	require.NoError(t, err, "No error expected during Resolve")
	require.NotNil(t, p1.c1, "Child1 must have been registered")
	require.NotNil(t, p1.c1.gc1, "Grandchild1 must have been registered")
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	g := New()

	for i := 0; i < 10; i++ {
		go func() {
			require.NoError(t, g.Register(NewGrandchild1))

			var gc1 *Grandchild1
			require.NoError(t, g.Resolve(&gc1))
		}()
	}
}

func TestCycles(t *testing.T) {
	t.Parallel()
	g := New()

	type Type1 interface{}
	type Type2 interface{}
	type Type3 interface{}
	c1 := func(t2 Type2) Type1 { return nil }
	c2 := func(t3 Type3) Type2 { return nil }
	c3 := func(t1 Type1) Type3 { return nil }

	require.NoError(t, g.Register(c1))
	require.NoError(t, g.Register(c2))

	err := g.Register(c3)
	require.Contains(t, err.Error(), "unable to register dig.Type3")
	require.Contains(t, err.Error(), "cycle")
}

func TestResolveAll(t *testing.T) {
	t.Parallel()
	g := New()

	err := g.RegisterAll(
		NewGrandchild1,
		NewChild1,
		NewParent1,
	)
	require.NoError(t, err)

	var p1 *Parent1
	var p2 *Parent1
	var p3 *Parent1
	var p4 *Parent1

	err = g.ResolveAll(&p1, &p2, &p3, &p4)
	require.NoError(t, err, "Did not expect error on resolve all")
	require.Equal(t, p1.name, "Parent1")
	require.True(t, p1 == p2 && p2 == p3 && p3 == p4, "All pointers must be equal")
}

func TestEmptyAfterReset(t *testing.T) {
	t.Parallel()
	g := New()

	require.NoError(t, g.Register(NewGrandchild1))

	var first *Grandchild1
	require.NoError(t, g.Resolve(&first), "No error expected during first Resolve")
	g.Reset()
	require.Contains(t, g.Resolve(&first).Error(), "not registered")
}
