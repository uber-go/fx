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

func TestWithMiddlewares(t *testing.T) {
	fakeMiddleware1 := fakeMiddleware()
	fakeMiddleware2 := fakeMiddleware()
	tests := []struct {
		desc   string
		mi     service.ModuleCreateInfo
		input  []Middleware
		expect []Middleware
	}{
		{
			desc:   "TestWithOneMiddleware",
			mi:     service.ModuleCreateInfo{},
			input:  []Middleware{fakeMiddleware1},
			expect: []Middleware{fakeMiddleware1},
		},
		{
			desc:   "TestWithTwoMiddlewares",
			mi:     service.ModuleCreateInfo{},
			input:  []Middleware{fakeMiddleware1, fakeMiddleware2},
			expect: []Middleware{fakeMiddleware1, fakeMiddleware2},
		},
		{
			desc: "TestHasOneWithOneMiddleware",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1},
				},
			},
			input:  []Middleware{fakeMiddleware2},
			expect: []Middleware{fakeMiddleware1, fakeMiddleware2},
		},
		{
			desc: "TestHasOneWithNilMiddleware",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1},
				},
			},
			input:  nil,
			expect: []Middleware{fakeMiddleware1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			opt := WithMiddlewares(tt.input...)
			assert.NoError(t, opt(&tt.mi))
			assert.Equal(t, len(tt.expect), len(middlewaresFromCreateInfo(tt.mi)))
		})
	}
}

func TestWithMiddlewaresTwice(t *testing.T) {
	fakeMiddleware1 := fakeMiddleware()
	fakeMiddleware2 := fakeMiddleware()
	tests := []struct {
		desc   string
		mi     service.ModuleCreateInfo
		input  []Middleware
		expect []Middleware
	}{
		{
			desc:   "TestWithOneMiddlewareTwice",
			mi:     service.ModuleCreateInfo{},
			input:  []Middleware{fakeMiddleware1},
			expect: []Middleware{fakeMiddleware1, fakeMiddleware1},
		},
		{
			desc:   "TestWithTwoMiddlewaresTwice",
			mi:     service.ModuleCreateInfo{},
			input:  []Middleware{fakeMiddleware1, fakeMiddleware2},
			expect: []Middleware{fakeMiddleware1, fakeMiddleware2, fakeMiddleware1, fakeMiddleware2},
		},
		{
			desc: "TestHasOneWithOneMiddlewareTwice",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1},
				},
			},
			input:  []Middleware{fakeMiddleware2},
			expect: []Middleware{fakeMiddleware1, fakeMiddleware2, fakeMiddleware2},
		},
		{
			desc: "TestHasOneWithNilMiddlewareTwice",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1},
				},
			},
			input:  nil,
			expect: []Middleware{fakeMiddleware1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			opt := WithMiddlewares(tt.input...)
			assert.NoError(t, opt(&tt.mi))

			opt = WithMiddlewares(tt.input...)
			assert.NoError(t, opt(&tt.mi))

			assert.Equal(t, len(tt.expect), len(middlewaresFromCreateInfo(tt.mi)))
		})
	}
}

func TestMiddlewaresFromCreateInfo(t *testing.T) {
	fakeMiddleware1 := fakeMiddleware()
	fakeMiddleware2 := fakeMiddleware()
	tests := []struct {
		desc        string
		mi          service.ModuleCreateInfo
		middlewares []Middleware
	}{
		{
			desc:        "TestEmptyItems",
			mi:          service.ModuleCreateInfo{},
			middlewares: nil,
		},
		{
			desc: "TestSometingElseInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					"somethingElse": []Middleware{fakeMiddleware1},
				},
			},
			middlewares: nil,
		},
		{
			desc: "TestOneMiddlewareInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1},
				},
			},
			middlewares: []Middleware{fakeMiddleware1},
		},
		{
			desc: "TestTwoMiddlewaresInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_middlewareKey: []Middleware{fakeMiddleware1, fakeMiddleware2},
				},
			},
			middlewares: []Middleware{fakeMiddleware1, fakeMiddleware2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			fs := middlewaresFromCreateInfo(tt.mi)
			assert.Equal(t, len(tt.middlewares), len(fs))
		})
	}
}
