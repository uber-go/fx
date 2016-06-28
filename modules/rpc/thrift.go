package rpc

import (
	"fmt"
	"time"

	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/metrics"
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/yarpc/yarpc-go/encoding/thrift"
)

type CreateThriftServiceFunc func(service *core.Service) (thrift.Service, error)

func ThriftModule(name string, moduleRoles []string, hookup CreateThriftServiceFunc) core.ModuleCreateFunc {
	return func(svc *core.Service) ([]core.Module, error) {
		if mod, err := newYarpcThriftModule(name, svc, moduleRoles, hookup); err != nil {
			return nil, err
		} else {
			return []core.Module{mod}, nil
		}
	}
}

func newYarpcThriftModule(name string, service *core.Service, roles []string, createService CreateThriftServiceFunc) (*YarpcModule, error) {

	svc, err := createService(service)
	if err != nil {
		return nil, err
	}

	reg := func(mod *YarpcModule) {
		wrappedService := serviceWrapper{mod: mod, service: svc}
		thrift.Register(mod.rpc, wrappedService)
	}
	return newYarpcModule(name, service, roles, reg)
}

type serviceWrapper struct {
	mod      *YarpcModule
	service  thrift.Service
	handlers map[string]thrift.Handler
	callback thrift.HandlerFunc
}

func (sw serviceWrapper) Name() string {
	return sw.service.Name()
}

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
		func(req *thrift.ReqMeta, body wire.Value) (thrift.Response, error) {

			data := map[string]string{
				// todo, what's the right tchannel header name?
				metrics.TrafficCorrelationID: req.Headers["cid"],
			}
			key := fmt.Sprintf("rpc.%s.%s", sw.service.Name(), name)
			tracker := reporter.Start(key, data, 90*time.Second)
			res, err := handler.Handle(req, body)
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
