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
	type t1 struct{}
	type t2 struct{}

	t.Run("populate nothing", func(t *testing.T) {
		app := fxtest.New(t,
			Provide(func() *t1 { panic("should not be called ") }),
			Populate(),
		)
		require.NoError(t, app.Err())
	})

	t.Run("populate single", func(t *testing.T) {
		var v1 *t1
		app := fxtest.New(t,
			Provide(func() *t1 { return &t1{} }),
			Populate(&v1),
		)
		require.NoError(t, app.Err())
		require.NotNil(t, v1, "did not populate value")
	})

	t.Run("populate interface", func(t *testing.T) {
		var reader io.Reader
		app := fxtest.New(t,
			Provide(func() io.Reader { return strings.NewReader("hello world") }),
			Populate(&reader),
		)
		require.NoError(t, app.Err())

		bs, err := ioutil.ReadAll(reader)
		require.NoError(t, err, "Failed to use populated io.Reader")
		assert.Equal(t, "hello world", string(bs), "Unexpected reader")
	})

	t.Run("populate multiple inline values", func(t *testing.T) {
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
		require.NoError(t, app.Err())
		require.NotNil(t, v1, "did not populate argument 1")
		require.NotNil(t, v2, "did not populate argument 2")
	})

	t.Run("populate fx.In struct", func(t *testing.T) {
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
		require.NoError(t, app.Err())
		require.NotNil(t, targets.V1, "did not populate field 1")
		require.NotNil(t, targets.V2, "did not populate field 2")
	})
}

func TestPopulateErrors(t *testing.T) {
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
			wantErr: "isn't in the container",
		},
		{
			msg:     "function",
			opt:     Populate(fn),
			wantErr: "not a pointer",
		},
		{
			msg:     "function pointer",
			opt:     Populate(&fn),
			wantErr: "isn't in the container",
		},
		{
			msg:     "invalid last argument",
			opt:     Populate(&v, t1{}),
			wantErr: "argument 2 is not a pointer type",
		},
		{
			msg:     "nil argument",
			opt:     Populate(&v, nil, &v),
			wantErr: "argument 2 is nil",
		},
	}

	for _, tt := range tests {
		app := fxtest.New(t,
			NopLogger,
			Provide(func() *t1 { return &t1{} }),

			tt.opt,
		)
		require.Error(t, app.Err())
		assert.Contains(t, app.Err().Error(), tt.wantErr)
	}
}
