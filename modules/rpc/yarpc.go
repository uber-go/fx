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
	"errors"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	errs "github.com/pkg/errors"
	"go.uber.org/fx/dig"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

// YARPCModule is an implementation of a core RPC module using YARPC.
// All the YARPC modules share the same dispatcher and middleware.
// Dispatcher will start when any created module calls Start().
// The YARPC team advised dispatcher to be a 'singleton' to control
// the lifecycle of all of the in/out bound traffic, so we will
// register it in a dig.Graph provided with options/default graph.
type YARPCModule struct {
	modules.ModuleBase
	config      yarpcConfig
	log         *zap.Logger
	statsClient *statsClient
	stateMu     sync.RWMutex
	isRunning   bool
	controller  *dispatcherController
}

var (
	_dispatcherMu sync.Mutex

	// Function to create a dispatcher
	_dispatcherFn = defaultYARPCDispatcher

	// Function to start a dispatcher
	_starterFn = defaultYARPCStarter

	_ service.Module = &YARPCModule{}
)

type transports struct {
	inbounds []transport.Inbound
}

type yarpcConfig struct {
	modules.ModuleConfig

	inboundMiddleware       []middleware.UnaryInbound
	onewayInboundMiddleware []middleware.OnewayInbound

	transports transports
	Inbounds   []Inbound
}

// Inbound is a union that configures how to configure a single inbound.
type Inbound struct {
	TChannel *Address
	HTTP     *Address
}

func (i *Inbound) String() string {
	if i == nil {
		return ""
	}

	http := "none"
	if i.HTTP != nil {
		http = strconv.Itoa(i.HTTP.Port)
	}

	tchannel := "none"
	if i.TChannel != nil {
		tchannel = strconv.Itoa(i.TChannel.Port)
	}

	return fmt.Sprintf("Inbound:{HTTP: %s; TChannel: %s}", http, tchannel)
}

// Address is a struct that have a required port for tchannel/http transports.
// TODO(alsam) make it optional
type Address struct {
	Port int
}

type handlerWithDispatcher func(dispatcher *yarpc.Dispatcher) error

// Stores a collection of all module configs with a shared dispatcher
// and user handles to work with the dispatcher.
type dispatcherController struct {
	// sync configs
	sync.RWMutex

	// idempotent start and stop
	start      sync.Once
	stop       sync.Once
	stopError  error
	startError error

	configs    []*yarpcConfig
	handlers   []handlerWithDispatcher
	dispatcher yarpc.Dispatcher
}

// Adds the config to the controller.
func (c *dispatcherController) addConfig(config yarpcConfig) {
	c.Lock()
	defer c.Unlock()

	c.configs = append(c.configs, &config)
}

// Adds the config to the controller.
func (c *dispatcherController) appendHandler(handler handlerWithDispatcher) {
	c.Lock()
	defer c.Unlock()

	c.handlers = append(c.handlers, handler)
}

// Apply handlers to the dispatcher.
func (c *dispatcherController) applyHandlers() error {
	for _, h := range c.handlers {
		if err := h(&c.dispatcher); err != nil {
			return err
		}
	}

	return nil
}

// Adds the default middleware: context propagation and auth.
func (c *dispatcherController) addDefaultMiddleware(host service.Host, statsClient *statsClient) {
	cfg := yarpcConfig{
		inboundMiddleware: []middleware.UnaryInbound{
			contextInboundMiddleware{host, statsClient},
			panicInboundMiddleware{statsClient},
			authInboundMiddleware{host, statsClient},
		},
		onewayInboundMiddleware: []middleware.OnewayInbound{
			contextOnewayInboundMiddleware{host},
			panicOnewayInboundMiddleware{statsClient},
			authOnewayInboundMiddleware{host, statsClient},
		},
	}

	c.addConfig(cfg)
}

// Starts the dispatcher:
// 1. Add default middleware and merge all existing configs
// 2. Create a dispatcher
// 3. Call user handles to e.g. register transport.Procedures on the dispatcher
// 4. Start the dispatcher
//
// Once started the controller will not start the dispatcher again.
func (c *dispatcherController) Start(host service.Host, statsClient *statsClient) error {
	c.start.Do(func() {
		c.addDefaultMiddleware(host, statsClient)

		var cfg yarpc.Config
		var err error
		if cfg, err = c.mergeConfigs(host.Name()); err != nil {
			c.startError = err
			return
		}

		_dispatcherMu.Lock()
		defer _dispatcherMu.Unlock()

		var d *yarpc.Dispatcher
		if d, err = _dispatcherFn(host, cfg); err != nil {
			c.startError = err
			return
		}

		c.dispatcher = *d
		if err := c.applyHandlers(); err != nil {
			c.startError = err
			return
		}

		c.startError = _starterFn(&c.dispatcher)
	})

	return c.startError
}

// Return the result of the dispatcher Stop() on the first call.
// No-op on subsequent calls.
// TODO: update readme/docs/examples GFM(339)
func (c *dispatcherController) Stop() error {
	c.stop.Do(func() {
		c.stopError = c.dispatcher.Stop()
	})

	return c.stopError
}

// Merge all the YARPC configs in the collection: transports and middleware are going to be shared.
// The name comes from the first config in the collection and is the same among all configs.
func (c *dispatcherController) mergeConfigs(name string) (conf yarpc.Config, err error) {
	c.RLock()
	defer c.RUnlock()

	// Config collection should always have an additional config with the default middleware.
	if len(c.configs) <= 1 {
		return conf, errors.New("unable to merge empty configs")
	}

	conf.Name = name

	// Collect all Inbounds and middleware from all configs
	var inboundMiddleware []middleware.UnaryInbound
	var onewayInboundMiddleware []middleware.OnewayInbound
	for _, cfg := range c.configs {
		conf.Inbounds = append(conf.Inbounds, cfg.transports.inbounds...)
		inboundMiddleware = append(inboundMiddleware, cfg.inboundMiddleware...)
		onewayInboundMiddleware = append(onewayInboundMiddleware, cfg.onewayInboundMiddleware...)
	}

	// Build the inbound middleware
	conf.InboundMiddleware = yarpc.InboundMiddleware{
		Unary:  yarpc.UnaryInboundMiddleware(inboundMiddleware...),
		Oneway: yarpc.OnewayInboundMiddleware(onewayInboundMiddleware...),
	}

	return conf, nil
}

// Creates a new YARPC module and adds a common middleware to the global config collection.
// The first created module defines the service name.
func newYARPCModule(
	mi service.ModuleCreateInfo,
	reg CreateThriftServiceFunc,
	options ...modules.Option,
) (*YARPCModule, error) {
	name := "yarpc"
	if mi.Name != "" {
		name = mi.Name
	}

	module := &YARPCModule{
		ModuleBase: *modules.NewModuleBase(name, mi.Host, []string{}),
	}

	module.log = zap.L().With(zap.String("moduleName", name))
	for _, opt := range options {
		if err := opt(&mi); err != nil {
			return module, errs.Wrap(err, "unable to apply option to YARPC module")
		}
	}

	config := module.Host().Config().Scope("modules").Get(module.Name())
	if err := config.PopulateStruct(&module.config); err != nil {
		return nil, errs.Wrap(err, "can't read inbounds")
	}

	// Iterate over inbounds.
	transportsIn, err := prepareInbounds(module.config.Inbounds, mi.Host.Name())
	if err != nil {
		return nil, errs.Wrap(err, "can't process inbounds")
	}

	module.config.transports.inbounds = transportsIn
	module.config.inboundMiddleware = inboundMiddlewareFromCreateInfo(mi)
	module.config.onewayInboundMiddleware = onewayInboundMiddlewareFromCreateInfo(mi)

	// Try to resolve a controller first
	// TODO(alsam) use dig options when available, because we can overwrite the controller in case of multiple
	// modules registering a controller.
	if err := dig.Resolve(&module.controller); err != nil {

		// Try to register it then
		module.controller = &dispatcherController{}
		if errCr := dig.Register(module.controller); errCr != nil {
			return nil, errs.Wrap(errCr, "can't register a dispatcher controller")
		}

		// Register dispatcher
		if err := dig.Register(&module.controller.dispatcher); err != nil {
			return nil, errs.Wrap(err, "unable to register the dispatcher")
		}
	}

	module.controller.addConfig(module.config)
	module.controller.appendHandler(func(dispatcher *yarpc.Dispatcher) error {
		t, err := reg(mi.Host)
		if err != nil {
			return err
		}

		dispatcher.Register(t)
		return nil
	})

	module.log.Info("Module successfuly created", zap.Any("inbounds", module.config.Inbounds))

	return module, nil
}

// Iterate over all inbounds and prepare corresponding transports
func prepareInbounds(inbounds []Inbound, serviceName string) (transportsIn []transport.Inbound, err error) {
	transportsIn = make([]transport.Inbound, 0, 2*len(inbounds))
	for _, in := range inbounds {
		if h := in.HTTP; h != nil {
			transportsIn = append(
				transportsIn,
				http.NewTransport().NewInbound(fmt.Sprintf(":%d", h.Port)))
		}

		if t := in.TChannel; t != nil {
			chn, err := tch.NewChannelTransport(
				tch.ServiceName(serviceName),
				tch.ListenAddr(fmt.Sprintf(":%d", t.Port)))

			if err != nil {
				return nil, errs.Wrap(err, "can't create tchannel transport")
			}

			transportsIn = append(transportsIn, chn.NewInbound())
		}
	}

	return transportsIn, nil
}

// Start begins serving requests with YARPC.
func (m *YARPCModule) Start(readyCh chan<- struct{}) <-chan error {
	ret := make(chan error, 1)
	if m.IsRunning() {
		ret <- errors.New("module is already running")
		return ret
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	// TODO(alsam) allow services to advertise with a name separate from the host name.
	if err := m.controller.Start(m.Host(), m.statsClient); err != nil {
		ret <- errs.Wrap(err, "unable to start dispatcher")
		return ret
	}

	m.log.Info("Module started")

	m.isRunning = true
	readyCh <- struct{}{}
	ret <- nil
	return ret
}

// Stop shuts down the YARPC module: stops the dispatcher.
func (m *YARPCModule) Stop() error {
	if !m.IsRunning() {
		return nil
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	m.isRunning = false
	return m.controller.Stop()
}

// IsRunning returns whether a module is running
func (m *YARPCModule) IsRunning() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.isRunning
}

// DispatcherFn allows override a dispatcher creation, e.g. if it is embedded in another struct.
type DispatcherFn func(service.Host, yarpc.Config) (*yarpc.Dispatcher, error)

// RegisterDispatcher allows you to override the YARPC dispatcher registration
func RegisterDispatcher(dispatchFn DispatcherFn) {
	_dispatcherMu.Lock()
	defer _dispatcherMu.Unlock()
	_dispatcherFn = dispatchFn
}

func defaultYARPCDispatcher(_ service.Host, cfg yarpc.Config) (*yarpc.Dispatcher, error) {
	return yarpc.NewDispatcher(cfg), nil
}

// StarterFn overrides start for dispatcher, e.g. attach some metrics with start.
type StarterFn func(dispatcher *yarpc.Dispatcher) error

// RegisterStarter allows you to override function that starts a dispatcher.
func RegisterStarter(startFn StarterFn) {
	_dispatcherMu.Lock()
	defer _dispatcherMu.Unlock()
	_starterFn = startFn
}

func defaultYARPCStarter(dispatcher *yarpc.Dispatcher) error {
	return dispatcher.Start()
}
