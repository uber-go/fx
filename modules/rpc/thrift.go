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
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"github.com/pkg/errors"
	"go.uber.org/yarpc/api/transport"
)

var _setupMu sync.Mutex

// CreateThriftServiceFunc creates a Thrift service from a service host
type CreateThriftServiceFunc func(svc service.Host) ([]transport.Procedure, error)

// ThriftModule creates a Thrift Module from a service func
func ThriftModule(hookup CreateThriftServiceFunc, options ...modules.Option) service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		if mi.Name == "" {
			mi.Name = "rpc"
		}

		mod, err := newYARPCThriftModule(mi, hookup, options...)
		if err != nil {
			return nil, errors.Wrap(err, "unable to instantiate Thrift module")
		}

		return []service.Module{mod}, nil
	}
}

func newYARPCThriftModule(
	mi service.ModuleCreateInfo,
	createService CreateThriftServiceFunc,
	options ...modules.Option,
) (*YARPCModule, error) {
	registrants, err := createService(mi.Host)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create YARPC thrift handler")
	}

	reg := func(mod *YARPCModule) {
		_setupMu.Lock()
		defer _setupMu.Unlock()
		Dispatcher().Register(registrants)
	}

	return newYARPCModule(mi, reg, options...)
}
