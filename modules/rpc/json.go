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

package rpc

import (
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"go.uber.org/yarpc/encoding/json"
)

// CreateJSONRegistrantsFunc returns a slice of registrants from a service host
type CreateJSONRegistrantsFunc func(service service.Host) []json.Registrant

// JSONModule instantiates a core module from a registrant func
func JSONModule(hookup CreateJSONRegistrantsFunc, options ...modules.Option) service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod, err := newYarpcJSONModule(mi, hookup, options...)
		if err == nil {
			return []service.Module{mod}, nil
		}

		return nil, err
	}
}

func newYarpcJSONModule(mi service.ModuleCreateInfo, createService CreateJSONRegistrantsFunc, options ...modules.Option) (*YarpcModule, error) {
	reg := func(mod *YarpcModule) {
		procs := createService(mi.Host)

		if procs != nil {
			for _, proc := range procs {
				json.Register(mod.rpc, proc)
			}
		}
	}

	return newYarpcModule(mi, reg, options...)
}
