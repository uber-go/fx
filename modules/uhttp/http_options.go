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
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

const _middlewareKey = "uhttpServerMiddleware"

// WithMiddlewares adds middlewares to uhttp Module that will be applied to all incoming http requests.
func WithMiddlewares(ms ...Middleware) modules.Option {
	return func(mci *service.ModuleCreateInfo) error {
		middlewares := middlewaresFromCreateInfo(*mci)
		middlewares = append(middlewares, ms...)
		if mci.Items == nil {
			mci.Items = make(map[string]interface{})
		}
		mci.Items[_middlewareKey] = middlewares

		return nil
	}
}

func middlewaresFromCreateInfo(mci service.ModuleCreateInfo) []Middleware {
	if items, ok := mci.Items[_middlewareKey]; ok {
		// Intentionally panic if programmer adds non-middleware slice to the data
		return items.([]Middleware)
	}
	return nil
}
