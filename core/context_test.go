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

package core

import (
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"

	gcontext "context"
)

type _testStruct struct {
	data string
}

func TestContext_Simple(t *testing.T) {
	ctx := NewContext(gcontext.Background(), service.NullHost())
	testStruct := _testStruct{
		data: "hello",
	}
	ctx.SetResource("resource", testStruct)

	assert.NotNil(t, ctx.Resource("resource"))
	assert.Equal(t, "hello", ctx.Resource("resource").(_testStruct).data)
}

func TestContext_NilResource(t *testing.T) {
	ctx := NewContext(gcontext.Background(), service.NullHost())

	assert.Nil(t, ctx.Resource("resource"))
}

func TestContext_HostAccess(t *testing.T) {
	ctx := NewContext(gcontext.Background(), service.NullHost())
	assert.NotNil(t, ctx.Config())
	assert.Equal(t, "dummy", ctx.Name())
}
