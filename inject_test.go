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

package fx

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

func TestInject(t *testing.T) {
	type type1 struct{}
	type type2 struct{}
	type type3 struct{}

	t.Run("Failures", func(t *testing.T) {
		tests := []interface{}{
			3,
			func() {},
			struct{}{},
			struct{ Foo *bytes.Buffer }{},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("%T", tt), func(t *testing.T) {
				app := New(
					Provide(func() *bytes.Buffer { return &bytes.Buffer{} }),
					Inject(&tt),
				)

				err := app.Start(context.Background())
				require.Error(t, err, "expected failure")
				require.Contains(t, err.Error(), "Inject expected a pointer to a struct")
			})
		}
	})

	t.Run("Empty", func(t *testing.T) {
		new1 := func() *type1 { panic("new1 must not be called") }
		new2 := func() *type2 { panic("new2 must not be called") }

		var out struct{}
		app := New(
			Provide(new1, new2),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "start failed")
	})

	t.Run("StructIsInjected", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		var gave2 *type2
		new2 := func() *type2 {
			gave2 = &type2{}
			return gave2
		}

		var out struct {
			T1 *type1
			T2 *type2
		}

		app := New(
			Provide(new1, new2),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.NotNil(t, out.T2, "T2 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
		assert.True(t, gave2 == out.T2, "T2 must match")
	})

	t.Run("EmbeddedExportedField", func(t *testing.T) {
		type T1 struct{}

		var gave1 *T1
		new1 := func() *T1 {
			gave1 = &T1{}
			return gave1
		}

		var out struct{ *T1 }

		app := New(
			Provide(new1),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
	})

	t.Run("EmbeddedUnexportedField", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		var out struct{ *type1 }

		app := New(
			Provide(new1),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.type1, "type1 must not be nil")
		assert.True(t, gave1 == out.type1, "type1 must match")
	})

	t.Run("DuplicateFields", func(t *testing.T) {
		var gave *type1
		new1 := func() *type1 {
			require.Nil(t, gave, "gave must be nil")
			gave = &type1{}
			return gave
		}

		var out struct {
			X *type1
			Y *type1
		}

		app := New(
			Provide(new1),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.X, "X must not be nil")
		assert.NotNil(t, out.Y, "Y must not be nil")
		assert.True(t, gave == out.X, "X must match")
		assert.True(t, gave == out.Y, "Y must match")
	})

	t.Run("SkipsUnexported", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		new2 := func() *type2 { panic("new2 must not be called") }

		var gave3 *type3
		new3 := func() *type3 {
			gave3 = &type3{}
			return gave3
		}

		var out struct {
			T1 *type1
			t2 *type2
			T3 *type3
		}

		app := New(
			Provide(new1, new2, new3),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.Nil(t, out.t2, "t2 must be nil")
		assert.NotNil(t, out.T3, "T3 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
		assert.True(t, gave3 == out.T3, "T3 must match")
	})

	t.Run("OverwritesExisting", func(t *testing.T) {
		type type1 struct{ value string }

		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{value: "foo"}
			return gave1
		}

		var out struct {
			T1 *type1
		}

		app := New(
			Provide(new1),
			Inject(&out),
		)

		old := &type1{value: "bar"}
		out.T1 = old

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.False(t, old == out.T1, "old value must have been overwritten")
		assert.True(t, gave1 == out.T1, "T1 must match")
	})

	t.Run("DoesNotZeroUnexported", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		new2 := func() *type2 { panic("new2 must not be called") }

		var out struct {
			T1 *type1
			t2 *type2
		}
		t2 := &type2{}
		out.t2 = t2

		app := New(
			Provide(new1, new2),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.NotNil(t, out.t2, "t2 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
		assert.True(t, t2 == out.t2, "t2 must match")
	})

	t.Run("NestedDigIn", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		var out struct {
			Result struct {
				dig.In

				T1 *type1
				T2 *type2 `optional:"true"`
			}
		}

		app := New(
			Provide(new1),
			Inject(&out),
		)

		require.NoError(t, app.Start(context.Background()), "failed to start")
		assert.NotNil(t, out.Result.T1, "T1 must not be nil")
		assert.Nil(t, out.Result.T2, "T2 must be nil")
		assert.True(t, gave1 == out.Result.T1, "T1 must match")
	})
}

func ExampleInject() {
	var target struct {
		Logger *log.Logger
	}

	app := New(
		Provide(func() *log.Logger { return log.New(os.Stdout, "", 0) }),
		Inject(&target),
	)

	app.Start(context.Background())
	target.Logger.Print("Injected!")

	// Output:
	// Injected!
}
