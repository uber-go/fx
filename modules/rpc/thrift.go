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
	"fmt"
	"time"

	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/uberfx/modules"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
	"golang.org/x/net/context"
)

// CreateThriftServiceFunc creates a Thrift service from a service host
type CreateThriftServiceFunc func(service core.ServiceHost) (thrift.Service, error)

// ThriftModule creates a Thrift Module from a service func
func ThriftModule(hookup CreateThriftServiceFunc, options ...modules.ModuleOption) core.ModuleCreateFunc {
	return func(mi core.ModuleCreateInfo) ([]core.Module, error) {
		mod, err := newYarpcThriftModule(mi, hookup, options...)
		if err != nil {
			return nil, err
		}

		return []core.Module{mod}, nil
	}
}

func newYarpcThriftModule(mi core.ModuleCreateInfo, createService CreateThriftServiceFunc, options ...modules.ModuleOption) (*YarpcModule, error) {

	svc, err := createService(mi.Host)
	if err != nil {
		return nil, err
	}

	reg := func(mod *YarpcModule) {
		wrappedService := serviceWrapper{mod: mod, service: svc}
		thrift.Register(mod.rpc, wrappedService)
	}
	return newYarpcModule(mi, reg, options...)
}

type serviceWrapper struct {
	mod      *YarpcModule
	service  thrift.Service
	handlers map[string]thrift.Handler
	callback thrift.HandlerFunc
}

// Name returns a service's name
func (sw serviceWrapper) Name() string {
	return sw.service.Name()
}

// Protocol returns a service's protocol
func (sw serviceWrapper) Protocol() protocol.Protocol {
	return sw.service.Protocol()
}

func (sw serviceWrapper) wrapHandler(name string, handler thrift.Handler) thrift.HandlerFunc {

	// I want to use YARPC middleware for this but it's not that helpful if I have to wrap each of the handlers
	// individually - basically the same as below.
	//
	// I want something like rpc.RegisterInterceptor(myInterceptor)
	reporter := sw.mod.Reporter()

	return thrift.HandlerFunc(
		func(ctx context.Context, req yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {

			data := map[string]string{}

			if cid, ok := req.Headers().Get("cid"); ok {
				// todo, what's the right tchannel header name?
				data[metrics.TrafficCorrelationID] = cid
			}

			key := fmt.Sprintf("rpc.%s.%s", sw.service.Name(), name)
			tracker := reporter.Start(key, data, 90*time.Second)
			res, err := handler.Handle(ctx, req, body)
			tracker.Finish("", res, err)
			return res, err
		},
	)
}

func (sw serviceWrapper) Handlers() map[string]thrift.Handler {
	if sw.handlers == nil {
		h := sw.service.Handlers()
		sw.handlers = make(map[string]thrift.Handler, len(h))

		for k, v := range h {
			sw.handlers[k] = sw.wrapHandler(k, v)
		}
	}
	return sw.handlers
}
