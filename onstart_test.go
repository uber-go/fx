// Copyright (c) 2019 Uber Technologies, Inc.
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

package fx

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnStart(t *testing.T) {
	type A string
	type B string

	type Params struct {
		In

		A A
		B B
	}

	var called []string

	// No arguments
	noArgs := func() {
		called = append(called, "noArgs")
	}
	hasArgs := func(a A, b B) {
		called = append(called, fmt.Sprintf("hasArgs(%v,%v)", a, b))
	}
	hasCtxAndArgs := func(ctx context.Context, a A) {
		called = append(called, fmt.Sprintf("hasCtxAndArgs(%v,%v)", ctx != nil, a))
	}
	hasParams := func(p Params) {
		called = append(called, fmt.Sprintf("hasParams(%v,%v)", p.A, p.B))
	}

	app := New(
		Provide(
			func() A { return "val-A" },
			func() B { return "val-B" },
		),

		OnStart(noArgs, hasArgs),
		OnStart(hasCtxAndArgs),
		OnStart(hasParams),
	)
	require.NoError(t, app.Err(), "Failed to create app")
	err := app.Start(context.Background())
	require.NoError(t, err)

	want := []string{
		"noArgs",
		"hasArgs(val-A,val-B)",
		"hasCtxAndArgs(true,val-A)",
		"hasParams(val-A,val-B)",
	}
	assert.Equal(t, want, called)
}
