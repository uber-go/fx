// Copyright (c) 2016 Uber Technologies, Inc.
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

package fxcontext

import (
	gcontext "context"
	"testing"

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	"github.com/stretchr/testify/assert"
)

func TestContext_HostAccess(t *testing.T) {
	ctx := New(gcontext.Background(), service.NullHost())
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Logger())
	assert.NotNil(t, ctx.Value(_fxContextStore))
}

func TestWithContext(t *testing.T) {
	gctx := gcontext.WithValue(gcontext.Background(), "key", "val")
	ctx := New(gctx, service.NullHost())
	assert.Equal(t, "val", ctx.Value("key"))

	gctx1 := gcontext.WithValue(gcontext.Background(), "key1", "val1")
	ctx = &Context{
		Context: gctx1,
	}
	assert.Equal(t, nil, ctx.Value("key"))
	assert.Equal(t, "val1", ctx.Value("key1"))
}

func TestWithContext_NilHost(t *testing.T) {
	ctx := New(gcontext.Background(), nil)
	assert.NotNil(t, ctx.Logger())
}

func TestContext_Convert(t *testing.T) {
	host := service.NullHost()
	ctx := New(gcontext.Background(), host)
	logger := ctx.Logger()
	assert.Equal(t, host.Logger(), logger)

	assertConvert(t, ctx, logger)
}

func assertConvert(t *testing.T, ctx gcontext.Context, logger ulog.Log) {
	fxctx := Context{ctx}
	assert.NotNil(t, fxctx.Logger())
	assert.Equal(t, fxctx.Logger(), logger)

	ctx = gcontext.WithValue(ctx, "key", nil)

	fxctx = Context{ctx}
	assert.NotNil(t, fxctx.Logger())
	assert.Equal(t, fxctx.Logger(), logger)
}
