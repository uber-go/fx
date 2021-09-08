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

package fx_test

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestPopulate(t *testing.T) {
	t.Parallel()

	// We make sure t1 has a size so when we compare pointers, 2 different
	// objects are not equal.
	type t1 struct {
		buf [1024]byte
	}
	type t2 struct{}

	_ = new(t1).buf // buf is unused

	t.Run("populate nothing", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			Provide(func() *t1 { panic("should not be called ") }),
			Populate(),
		)
		app.RequireStart().RequireStop()
	})

	t.Run("populate single", func(t *testing.T) {
		t.Parallel()

		var v1 *t1
		app := fxtest.New(t,
			Provide(func() *t1 { return &t1{} }),
			Populate(&v1),
		)
		app.RequireStart().RequireStop()
		require.NotNil(t, v1, "did not populate value")
	})

	t.Run("populate interface", func(t *testing.T) {
		t.Parallel()

		var reader io.Reader
		app := fxtest.New(t,
			Provide(func() io.Reader { return strings.NewReader("hello world") }),
			Populate(&reader),
		)
		app.RequireStart().RequireStop()

		bs, err := ioutil.ReadAll(reader)
		require.NoError(t, err, "Failed to use populated io.Reader")
		assert.Equal(t, "hello world", string(bs), "Unexpected reader")
	})

	t.Run("populate multiple inline values", func(t *testing.T) {
		t.Parallel()

		var (
			v1 *t1
			v2 *t2
		)
		app := fxtest.New(t,
			Provide(func() *t1 { return &t1{} }),
			Provide(func() *t2 { return &t2{} }),

			Populate(&v1),
			Populate(&v2),
		)
		app.RequireStart().RequireStop()

		require.NotNil(t, v1, "did not populate argument 1")
		require.NotNil(t, v2, "did not populate argument 2")
	})

	t.Run("populate fx.In struct", func(t *testing.T) {
		t.Parallel()

		targets := struct {
			In

			V1 *t1
			V2 *t2
		}{}

		app := fxtest.New(t,
			Provide(func() *t1 { return &t1{} }),
			Provide(func() *t2 { return &t2{} }),

			Populate(&targets),
		)
		app.RequireStart().RequireStop()

		require.NotNil(t, targets.V1, "did not populate field 1")
		require.NotNil(t, targets.V2, "did not populate field 2")
	})

	t.Run("populate named field", func(t *testing.T) {
		t.Parallel()

		type result struct {
			Out

			V1 *t1 `name:"n1"`
			V2 *t1 `name:"n2"`
		}

		targets := struct {
			In

			V1 *t1 `name:"n1"`
			V2 *t1 `name:"n2"`
		}{}

		app := fxtest.New(t,
			Provide(func() result {
				return result{
					V1: &t1{},
					V2: &t1{},
				}
			}),

			Populate(&targets),
		)
		app.RequireStart().RequireStop()

		require.NotNil(t, targets.V1, "did not populate field 1")
		require.NotNil(t, targets.V2, "did not populate field 2")
		// Cannot use assert.Equal here as we want to compare pointers.
		assert.False(t, targets.V1 == targets.V2, "fields should be different")
	})

	t.Run("populate group", func(t *testing.T) {
		t.Parallel()

		type result struct {
			Out

			V1 *t1 `group:"g"`
			V2 *t1 `group:"g"`
		}

		targets := struct {
			In

			Group []*t1 `group:"g"`
		}{}

		app := fxtest.New(t,
			Provide(func() result {
				return result{
					V1: &t1{},
					V2: &t1{},
				}
			}),

			Populate(&targets),
		)
		app.RequireStart().RequireStop()

		require.Len(t, targets.Group, 2, "Expected group to have 2 values")
		require.NotNil(t, targets.Group[0], "did not populate group value 1")
		require.NotNil(t, targets.Group[1], "did not populate group value 2")
		// Cannot use assert.Equal here as we want to compare pointers.
		assert.False(t, targets.Group[0] == targets.Group[1], "group values should be different")
	})
}

func TestPopulateErrors(t *testing.T) {
	t.Parallel()

	type t1 struct{}
	type container struct {
		In

		T1 t1
	}
	type containerNoIn struct {
		T1 t1
	}
	fn := func() {}
	var v *t1

	tests := []struct {
		msg     string
		opt     Option
		wantErr string
	}{
		{
			msg:     "inline value",
			opt:     Populate(t1{}),
			wantErr: "not a pointer",
		},
		{
			msg:     "container value",
			opt:     Populate(container{}),
			wantErr: "not a pointer",
		},
		{
			msg:     "container pointer without fx.In",
			opt:     Populate(&containerNoIn{}),
			wantErr: "missing type: fx_test.containerNoIn",
		},
		{
			msg:     "function",
			opt:     Populate(fn),
			wantErr: "not a pointer",
		},
		{
			msg:     "function pointer",
			opt:     Populate(&fn),
			wantErr: "missing type: func()",
		},
		{
			msg:     "invalid last argument",
			opt:     Populate(&v, t1{}),
			wantErr: "target 2 is not a pointer type",
		},
		{
			msg:     "nil argument",
			opt:     Populate(&v, nil, &v),
			wantErr: "target 2 is nil",
		},
	}

	for _, tt := range tests {
		app := NewForTest(t,
			NopLogger,
			Provide(func() *t1 { return &t1{} }),

			tt.opt,
		)
		require.Error(t, app.Err())
		assert.Contains(t, app.Err().Error(), tt.wantErr)
	}
}

func TestPopulateValidateApp(t *testing.T) {
	t.Parallel()

	type t1 struct{}
	type container struct {
		In

		T1 t1
	}
	type containerNoIn struct {
		T1 t1
	}
	fn := func() {}
	var v *t1

	tests := []struct {
		msg     string
		opts    []interface{}
		wantErr string
	}{
		{
			msg:     "inline value",
			opts:    []interface{}{t1{}},
			wantErr: "not a pointer",
		},
		{
			msg:     "container value",
			opts:    []interface{}{container{}},
			wantErr: "not a pointer",
		},
		{
			msg:     "container pointer without fx.In",
			opts:    []interface{}{&containerNoIn{}},
			wantErr: "missing type: fx_test.containerNoIn",
		},
		{
			msg:     "function",
			opts:    []interface{}{fn},
			wantErr: "not a pointer",
		},
		{
			msg:     "function pointer",
			opts:    []interface{}{&fn},
			wantErr: "missing type: func()",
		},
		{
			msg:     "invalid last argument",
			opts:    []interface{}{&v, t1{}},
			wantErr: "target 2 is not a pointer type",
		},
		{
			msg:     "nil argument",
			opts:    []interface{}{&v, nil, &v},
			wantErr: "target 2 is nil",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			t.Parallel()

			testOpts := []Option{
				NopLogger,
				Provide(func() *t1 { return &t1{} }),
				Populate(tt.opts...),
			}

			err := validateTestApp(t, testOpts...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
