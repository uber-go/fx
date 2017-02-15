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

func TestWithFilters(t *testing.T) {
	fakeFilter1 := fakeFilter()
	fakeFilter2 := fakeFilter()
	tests := []struct {
		desc   string
		mi     service.ModuleCreateInfo
		input  []Filter
		expect []Filter
	}{
		{
			desc:   "TestWithOneFilter",
			mi:     service.ModuleCreateInfo{},
			input:  []Filter{fakeFilter1},
			expect: []Filter{fakeFilter1},
		},
		{
			desc:   "TestWithTwoFilters",
			mi:     service.ModuleCreateInfo{},
			input:  []Filter{fakeFilter1, fakeFilter2},
			expect: []Filter{fakeFilter1, fakeFilter2},
		},
		{
			desc: "TestHasOneWithOneFilter",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_filterKey: []Filter{fakeFilter1},
				},
			},
			input:  []Filter{fakeFilter2},
			expect: []Filter{fakeFilter1, fakeFilter2},
		},
		{
			desc: "TestHasOneWithNilFilter",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_filterKey: []Filter{fakeFilter1},
				},
			},
			input:  nil,
			expect: []Filter{fakeFilter1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			opt := WithFilters(tt.input...)
			assert.NoError(t, opt(&tt.mi))
			assert.Equal(t, len(tt.expect), len(filtersFromCreateInfo(tt.mi)))
		})
	}
}

func TestFiltersFromCreateInfo(t *testing.T) {
	fakeFilter1 := fakeFilter()
	fakeFilter2 := fakeFilter()
	tests := []struct {
		desc    string
		mi      service.ModuleCreateInfo
		filters []Filter
	}{
		{
			desc:    "TestEmptyItems",
			mi:      service.ModuleCreateInfo{},
			filters: nil,
		},
		{
			desc: "TestSometingElseInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					"somethingElse": []Filter{fakeFilter1},
				},
			},
			filters: nil,
		},
		{
			desc: "TestOneFilterInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_filterKey: []Filter{fakeFilter1},
				},
			},
			filters: []Filter{fakeFilter1},
		},
		{
			desc: "TestTwoFiltersInItems",
			mi: service.ModuleCreateInfo{
				Items: map[string]interface{}{
					_filterKey: []Filter{fakeFilter1, fakeFilter2},
				},
			},
			filters: []Filter{fakeFilter1, fakeFilter2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			fs := filtersFromCreateInfo(tt.mi)
			assert.Equal(t, len(tt.filters), len(fs))
		})
	}
}
