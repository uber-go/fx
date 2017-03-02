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

package yarpc

import "go.uber.org/yarpc/api/middleware"

// ModuleOption is a function that configures module creation.
type ModuleOption func(*moduleOptions) error

type moduleOptions struct {
	unaryInbounds  []middleware.UnaryInbound
	onewayInbounds []middleware.OnewayInbound
}

// WithInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithInboundMiddleware(i ...middleware.UnaryInbound) ModuleOption {
	return func(moduleOptions *moduleOptions) error {
		moduleOptions.unaryInbounds = append(moduleOptions.unaryInbounds, i...)
		return nil
	}
}

// WithOnewayInboundMiddleware adds custom YARPC inboundMiddleware to the module
func WithOnewayInboundMiddleware(i ...middleware.OnewayInbound) ModuleOption {
	return func(moduleOptions *moduleOptions) error {
		moduleOptions.onewayInbounds = append(moduleOptions.onewayInbounds, i...)
		return nil
	}
}
