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

package uhttp

import (
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
)

func TestWithInboundMiddleware(t *testing.T) {
	fakeInbound1 := fakeInbound()
	fakeInbound2 := fakeInbound()
	tests := []struct {
		desc   string
		mi     service.ModuleCreateInfo
		input  []InboundMiddleware
		expect []InboundMiddleware
	}{
		{
			desc:   "TestWithOneInboundMiddleware",
			mi:     service.ModuleCreateInfo{},
			input:  []InboundMiddleware{fakeInbound1},
			expect: []InboundMiddleware{fakeInbound1},
		},
		{
			desc:   "TestWithTwoInboundMiddleware",
			mi:     service.ModuleCreateInfo{},
			input:  []InboundMiddleware{fakeInbound1, fakeInbound2},
			expect: []InboundMiddleware{fakeInbound1, fakeInbound2},
		},
		{
			desc: "TestHasOneWithOneInboundMiddleware",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1},
				},
			},
			input:  []InboundMiddleware{fakeInbound2},
			expect: []InboundMiddleware{fakeInbound1, fakeInbound2},
		},
		{
			desc: "TestHasOneWithNilInboundMiddleware",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1},
				},
			},
			input:  nil,
			expect: []InboundMiddleware{fakeInbound1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			opt := WithInboundMiddleware(tt.input...)
			assert.NoError(t, opt(&tt.mi))
			assert.Equal(t, len(tt.expect), len(inboundMiddlewareFromCreateInfo(tt.mi)))
		})
	}
}

func TestWithInboundMiddlewareTwice(t *testing.T) {
	fakeInbound1 := fakeInbound()
	fakeInbound2 := fakeInbound()
	tests := []struct {
		desc   string
		mi     service.ModuleCreateInfo
		input  []InboundMiddleware
		expect []InboundMiddleware
	}{
		{
			desc:   "TestWithOneInboundMiddlewareTwice",
			mi:     service.ModuleCreateInfo{},
			input:  []InboundMiddleware{fakeInbound1},
			expect: []InboundMiddleware{fakeInbound1, fakeInbound1},
		},
		{
			desc:   "TestWithTwoInboundMiddlewareTwice",
			mi:     service.ModuleCreateInfo{},
			input:  []InboundMiddleware{fakeInbound1, fakeInbound2},
			expect: []InboundMiddleware{fakeInbound1, fakeInbound2, fakeInbound1, fakeInbound2},
		},
		{
			desc: "TestHasOneWithOneInboundMiddlewareTwice",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1},
				},
			},
			input:  []InboundMiddleware{fakeInbound2},
			expect: []InboundMiddleware{fakeInbound1, fakeInbound2, fakeInbound2},
		},
		{
			desc: "TestHasOneWithNilInboundMiddlewareTwice",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1},
				},
			},
			input:  nil,
			expect: []InboundMiddleware{fakeInbound1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			opt := WithInboundMiddleware(tt.input...)
			assert.NoError(t, opt(&tt.mi))

			opt = WithInboundMiddleware(tt.input...)
			assert.NoError(t, opt(&tt.mi))

			assert.Equal(t, len(tt.expect), len(inboundMiddlewareFromCreateInfo(tt.mi)))
		})
	}
}

func TestInboundMiddlewareFromCreateInfo(t *testing.T) {
	fakeInbound1 := fakeInbound()
	fakeInbound2 := fakeInbound()
	tests := []struct {
		desc       string
		mi         service.ModuleCreateInfo
		middleware []InboundMiddleware
	}{
		{
			desc:       "TestEmptyItems",
			mi:         service.ModuleCreateInfo{},
			middleware: nil,
		},
		{
			desc: "TestSometingElseInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					"somethingElse": []InboundMiddleware{fakeInbound1},
				},
			},
			middleware: nil,
		},
		{
			desc: "TestOneInboundMiddlewareInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1},
				},
			},
			middleware: []InboundMiddleware{fakeInbound1},
		},
		{
			desc: "TestTwoInboundMiddlewareInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []InboundMiddleware{fakeInbound1, fakeInbound2},
				},
			},
			middleware: []InboundMiddleware{fakeInbound1, fakeInbound2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			fs := inboundMiddlewareFromCreateInfo(tt.mi)
			assert.Equal(t, len(tt.middleware), len(fs))
		})
	}
}
