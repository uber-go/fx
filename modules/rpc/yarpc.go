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
	"errors"
	"fmt"
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	errs "github.com/pkg/errors"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	tch "go.uber.org/yarpc/transport/tchannel"
)

// YarpcModule is an implementation of a core module using YARPC
type YarpcModule struct {
	modules.ModuleBase
	rpc                      *yarpc.Dispatcher
	register                 registerServiceFunc
	config                   yarpcConfig
	log                      ulog.Log
	stateMu                  sync.RWMutex
	inboundMiddlewares       []middleware.UnaryInbound
	onewayInboundMiddlewares []middleware.OnewayInbound
}

var (
	_dispatcherFn = defaultYarpcDispatcher

	_ service.Module = &YarpcModule{}
)

type registerServiceFunc func(module *YarpcModule)

// RPCModuleType represents the type of an RPC module
const RPCModuleType = "rpc"

type yarpcConfig struct {
	modules.ModuleConfig
	Bind          string `yaml:"bind"`
	AdvertiseName string `yaml:"advertiseName"`
}

func newYarpcModule(
	mi service.ModuleCreateInfo,
	reg registerServiceFunc,
	options ...modules.Option,
) (*YarpcModule, error) {
	cfg := &yarpcConfig{
		AdvertiseName: mi.Host.Name(),
		Bind:          ":0",
	}

	name := "yarpc"
	if mi.Name != "" {
		name = mi.Name
	}

	module := &YarpcModule{
		ModuleBase: *modules.NewModuleBase(RPCModuleType, name, mi.Host, []string{}),
		register:   reg,
		config:     *cfg,
	}

	options = append([]modules.Option{WithInboundMiddleware(fxContextInboundMiddleware{
		Host: mi.Host,
	}),
		WithInboundMiddleware(authInboundMiddleware{
			Host: mi.Host,
		}),
		WithOnewayInboundMiddleware(fxContextOnewayInboundMiddleware{
			Host: mi.Host,
		}),
		WithOnewayInboundMiddleware(authOnewayInboundMiddleware{
			Host: mi.Host,
		}),
	}, options...)

	module.log = ulog.Logger().With("moduleName", name)
	for _, opt := range options {
		if err := opt(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", opt)
			return module, errs.Wrap(err, "unable to apply option to YARPC module")
		}
	}

	err := module.Host().Config().Get(fmt.Sprintf("modules.%s", module.Name())).PopulateStruct(cfg)
	// found values, update module
	module.config = *cfg

	module.inboundMiddlewares = inboundMiddlewaresFromCreateInfo(mi)
	module.onewayInboundMiddlewares = onewayInboundMiddlewaresFromCreateInfo(mi)

	return module, err
}

// Initialize sets up a YARPC-backed module
func (m *YarpcModule) Initialize(service service.Host) error {
	return nil
}

// Start begins serving requests over YARPC
func (m *YarpcModule) Start(readyCh chan<- struct{}) <-chan error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	interceptor := yarpc.UnaryInboundMiddleware(m.inboundMiddlewares...)
	onewayInterceptor := yarpc.OnewayInboundMiddleware(m.onewayInboundMiddlewares...)

	tchTransport, err := tch.NewChannelTransport(
		tch.ServiceName(m.config.AdvertiseName),
		tch.ListenAddr(m.config.Bind),
	)
	ret := make(chan error, 1)
	if err != nil {
		ret <- errors.New("Unable to create TChannel transport " + err.Error())
		return ret
	}

	m.rpc, err = _dispatcherFn(m.Host(), yarpc.Config{
		Name: m.config.AdvertiseName,
		Inbounds: []transport.Inbound{
			tchTransport.NewInbound(),
		},
		InboundMiddleware: yarpc.InboundMiddleware{
			Unary:  interceptor,
			Oneway: onewayInterceptor,
		},
		Tracer: m.Tracer(),
	})

	if err != nil {
		ret <- err
		return ret
	}

	m.register(m)
	// TODO update log object to be accessed via context.Context #74
	m.log.Info("Service started", "service", m.config.AdvertiseName, "port", m.config.Bind)

	ret <- m.rpc.Start()
	readyCh <- struct{}{}
	return ret
}

// Stop shuts down a YARPC module
func (m *YarpcModule) Stop() error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	if m.rpc != nil {
		err := m.rpc.Stop()
		m.rpc = nil
		return err
	}
	return nil
}

// IsRunning returns whether a module is running
func (m *YarpcModule) IsRunning() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.rpc != nil
}

type yarpcDispatcherFn func(service.Host, yarpc.Config) (*yarpc.Dispatcher, error)

// RegisterDispatcher allows you to override the YARPC dispatcher registration
func RegisterDispatcher(dispatchFn yarpcDispatcherFn) {
	_dispatcherFn = dispatchFn
}

func defaultYarpcDispatcher(_ service.Host, cfg yarpc.Config) (*yarpc.Dispatcher, error) {
	return yarpc.NewDispatcher(cfg), nil
}
