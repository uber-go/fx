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

package rpc

import (
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
)

func TestWithInboundMiddleware_OK(t *testing.T) {
	opt := WithInboundMiddleware(middleware.NopUnaryInbound)
	mc := &service.ModuleCreateInfo{
		Items: make(map[string]interface{}),
	}

	require.NoError(t, opt(mc))
	assert.Equal(t, 1, len(inboundMiddlewaresFromCreateInfo(*mc)))
}

func TestWithOnewayInboundMiddleware_OK(t *testing.T) {
	opt := WithOnewayInboundMiddleware(middleware.NopOnewayInbound)
	mc := &service.ModuleCreateInfo{
		Items: make(map[string]interface{}),
	}
	require.NoError(t, opt(mc))
	assert.Equal(t, 1, len(onewayInboundMiddlewaresFromCreateInfo(*mc)))
}

func TestWithInboundMiddleware_PanicsBadData(t *testing.T) {
	opt := WithInboundMiddleware(middleware.NopUnaryInbound)
	mc := &service.ModuleCreateInfo{
		Items: map[string]interface{}{
			_interceptorKey: "foo",
		},
	}
	assert.Panics(t, func() {
		opt(mc)
	})
}

func TestWithOnewayInboundMiddleware_PanicsBadData(t *testing.T) {
	opt := WithOnewayInboundMiddleware(middleware.NopOnewayInbound)
	mc := &service.ModuleCreateInfo{
		Items: map[string]interface{}{
			_onewayInterceptorKey: "foo",
		},
	}
	assert.Panics(t, func() {
		opt(mc)
	})
}
