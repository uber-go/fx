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

package modules

import "github.com/uber-go/uberfx/core"

// ModuleOption is a function that configures module creation
type ModuleOption func(core.ModuleCreateInfo) error

// WithName is an option to set a module name
func WithName(name string) ModuleOption {
	return func(mi core.ModuleCreateInfo) error {
		mi.Name = name
		return nil
	}
}

// WithRoles is an option to set module roles
func WithRoles(roles ...string) ModuleOption {
	return func(mi core.ModuleCreateInfo) error {
		// if mb := findModuleInfo(module); mb != nil {
		// 	mb.roles = roles
		// }
		mi.Roles = roles
		return nil
	}
}

// func findModuleInfo(module core.Module) *ModuleBase {
// 	if val, ok := util.FindField(module, nil, reflect.TypeOf(ModuleBase{})); ok {
// 		mb := reflect.Indirect(val).Interface().(ModuleBase)
// 		return &mb
// 	}
// 	return nil
// }
