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

package service

import "github.com/pkg/errors"

type moduleCreateFuncOptionsPair struct {
	moduleCreateFunc ModuleCreateFunc
	options          []ModuleOption
}

// A Builder is a helper to create a service
type Builder struct {
	options []Option
	modules []*moduleCreateFuncOptionsPair
}

// WithModule is a helper to create a service without any options
func WithModule(module ModuleCreateFunc, options ...ModuleOption) *Builder {
	b := &Builder{}
	return b.WithModule(module, options...)
}

// WithModule adds the given module to the service
func (b *Builder) WithModule(module ModuleCreateFunc, options ...ModuleOption) *Builder {
	b.modules = append(b.modules, &moduleCreateFuncOptionsPair{module, options})
	return b
}

// WithOptions adds service Options to the builder
func (b *Builder) WithOptions(options ...Option) *Builder {
	b.options = append(b.options, options...)
	return b
}

// Build returns the service, or any errors encountered during build phase.
func (b *Builder) Build() (Manager, error) {
	svc, err := newManager(b)
	if err != nil {
		return nil, errors.Wrap(err, "service instantiation failed")
	}
	return svc, nil
}
