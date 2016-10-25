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
	"sync"
	"time"

	"golang.org/x/net/context"

	"go.uber.org/fx/core/config"
	"go.uber.org/fx/core/metrics"
	"go.uber.org/fx/core/ulog"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"
	tch "go.uber.org/yarpc/transport/tchannel"
)

// YarpcModule is an implementation of a core module using YARPC
type YarpcModule struct {
	modules.ModuleBase
	rpc      yarpc.Dispatcher
	register registerServiceFunc
	config   yarpcConfig
	log      ulog.Log
	stateMu  sync.RWMutex
}

var _ service.Module = &YarpcModule{}

type registerServiceFunc func(module *YarpcModule)

// RPCModuleType represents the type of an RPC module
const RPCModuleType = "rpc"

type yarpcConfig struct {
	modules.ModuleConfig
	Bind          string `yaml:"bind"`
	AdvertiseName string `yaml:"advertise_name"`
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

	reporter := &metrics.LoggingReporter{Prefix: mi.Host.Name()}

	module := &YarpcModule{
		ModuleBase: *modules.NewModuleBase(RPCModuleType, name, mi.Host, reporter, []string{}),
		register:   reg,
		config:     *cfg,
	}

	module.log = ulog.Logger().With("moduleName", name)
	for _, opt := range options {
		if err := opt(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", opt)
			return module, err
		}
	}

	if config.Global().GetValue(fmt.Sprintf("modules.%s", module.Name())).PopulateStruct(cfg) {
		// found values, update module
		module.config = *cfg
	}

	return module, nil
}

// Initialize sets up a YAPR-backed module
func (m *YarpcModule) Initialize(service service.Host) error {
	return nil
}

// Start begins serving requests over YARPC
func (m *YarpcModule) Start(readyCh chan<- struct{}) <-chan error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	channel, err := tchannel.NewChannel(m.config.AdvertiseName, nil)
	if err != nil {
		m.log.Fatal("Unable to create TChannel", "error", err)
	}

	m.rpc = yarpc.NewDispatcher(yarpc.Config{
		Name: m.config.AdvertiseName,
		Inbounds: []transport.Inbound{
			tch.NewInbound(channel, tch.ListenAddr(m.config.Bind)),
		},
		Interceptor: m.makeInterceptor(),
	})

	m.register(m)
	ret := make(chan error, 1)
	// TODO update log object to be accessed via context.Context #74
	m.log.Info("Service started", "service", m.config.AdvertiseName, "port", m.config.Bind)

	ret <- m.rpc.Start()
	readyCh <- struct{}{}
	return ret
}

func (m *YarpcModule) makeInterceptor() transport.Interceptor {
	reporter := m.Reporter()
	return transport.InterceptorFunc(
		func(
			ctx context.Context,
			options transport.Options,
			request *transport.Request,
			response transport.ResponseWriter,
			h transport.Handler,
		) error {
			data := map[string]string{}

			if cid, ok := request.Headers.Get("cid"); ok {
				// todo, what's the right tchannel header name?
				data[metrics.TrafficCorrelationID] = cid
			}

			key := "rpc." + request.Procedure
			tracker := reporter.Start(key, data, 90*time.Second)
			err := h.Handle(ctx, options, request, response)
			tracker.Finish("", nil, err)
			return err
		},
	)
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
