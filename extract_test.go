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

package fx_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	. "go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

func TestExtract(t *testing.T) {
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
				app := fxtest.New(t,
					Provide(func() *bytes.Buffer { return &bytes.Buffer{} }),
					Extract(&tt),
				)
				err := app.Start(context.Background())
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Extract expected a pointer to a struct")
			})
		}
	})

	t.Run("Empty", func(t *testing.T) {
		new1 := func() *type1 { panic("new1 must not be called") }
		new2 := func() *type2 { panic("new2 must not be called") }

		var out struct{}
		app := fxtest.New(t,
			Provide(new1, new2),
			Extract(&out),
		)
		app.RequireStart().RequireStop()
	})

	t.Run("StructIsExtracted", func(t *testing.T) {
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

		app := fxtest.New(t,
			Provide(new1, new2),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
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

		app := fxtest.New(t,
			Provide(new1),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
	})

	t.Run("EmbeddedUnexportedField", func(t *testing.T) {
		new1 := func() *type1 { return &type1{} }
		var out struct{ *type1 }

		app := fxtest.New(t, Provide(new1), Extract(&out))
		defer app.RequireStart().RequireStop()

		// Unexported fields are left unchanged.
		assert.Nil(t, out.type1, "type1 must be nil")
	})

	t.Run("EmbeddedUnexportedFieldValue", func(t *testing.T) {
		type type4 struct{ foo string }
		new4 := func() type4 { return type4{"foo"} }
		var out struct{ type4 }

		app := fxtest.New(t, Provide(new4), Extract(&out))
		defer app.RequireStart().RequireStop()

		// Unexported fields are left unchanged.
		assert.NotEqual(t, "foo", out.type4.foo)
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

		app := fxtest.New(t,
			Provide(new1),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
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

		app := fxtest.New(t,
			Provide(new1, new2, new3),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.Nil(t, out.t2, "t2 must be nil")
		assert.NotNil(t, out.T3, "T3 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
		assert.True(t, gave3 == out.T3, "T3 must match")
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

		app := fxtest.New(t,
			Provide(new1, new2),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.NotNil(t, out.t2, "t2 must not be nil")
		assert.True(t, gave1 == out.T1, "T1 must match")
		assert.True(t, t2 == out.t2, "t2 must match")
	})

	t.Run("TopLevelDigIn", func(t *testing.T) {
		var out struct{ dig.In }
		app := fxtest.New(t, Extract(&out))
		defer app.RequireStart().RequireStop()
	})

	t.Run("TopLevelFxIn", func(t *testing.T) {
		new1 := func() *type1 { panic("new1 must not be called") }
		new2 := func() *type2 { panic("new2 must not be called") }

		var out struct{ In }
		app := fxtest.New(t,
			Provide(new1, new2),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
	})

	t.Run("NestedFxIn", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		var out struct {
			Result struct {
				In

				T1 *type1
				T2 *type2 `optional:"true"`
			}
		}

		app := fxtest.New(t,
			Provide(new1),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
		assert.NotNil(t, out.Result.T1, "T1 must not be nil")
		assert.Nil(t, out.Result.T2, "T2 must be nil")
		assert.True(t, gave1 == out.Result.T1, "T1 must match")
	})

	t.Run("FurtherNestedFxIn", func(t *testing.T) {
		var out struct {
			In

			B struct {
				In

				C int
			}
		}

		app := fxtest.New(t,
			Provide(func() int { return 42 }),
			Extract(&out),
		)
		defer app.RequireStart().RequireStop()
		assert.Equal(t, 42, out.B.C, "B.C must match")
	})

	t.Run("FieldsCanBeOptional", func(t *testing.T) {
		var gave1 *type1
		new1 := func() *type1 {
			gave1 = &type1{}
			return gave1
		}

		var out struct {
			T1 *type1
			T2 *type2 `optional:"true"`
		}

		app := fxtest.New(t,
			Provide(new1),
			Extract(&out),
		)

		defer app.RequireStart().RequireStop()
		assert.NotNil(t, out.T1, "T1 must not be nil")
		assert.Nil(t, out.T2, "T2 must be nil")

		assert.True(t, gave1 == out.T1, "T1 must match")
	})
}

func ExampleExtract() {
	var target struct {
		Logger *log.Logger
	}

	app := New(
		Provide(func() *log.Logger { return log.New(os.Stdout, "", 0) }),
		Extract(&target),
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	target.Logger.Print("Extracted!")

	// Output:
	// Extracted!
}
