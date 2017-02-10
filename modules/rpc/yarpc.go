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
	"sync"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/modules/rpc/internal/stats"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	errs "github.com/pkg/errors"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	tch "go.uber.org/yarpc/transport/tchannel"
)

// YARPCModule is an implementation of a core RPC module using YARPC.
// All the YARPC modules share the same dispatcher and middleware.
// Dispatcher will start when any created module calls Start().
type YARPCModule struct {
	modules.ModuleBase
	register  registerServiceFunc
	config    yarpcConfig
	log       ulog.Log
	stateMu   sync.RWMutex
	isRunning bool
}

var (
	_dispatcherFn = defaultYARPCDispatcher
	_dispatcherMu sync.Mutex

	_ service.Module = &YARPCModule{}

	// Controller represents a collection of all YARPC configs
	// that are stored together to create a shared dispatcher.
	// The YARPC team advised it to be a 'singleton' to control
	// the lifecycle of all of the in/out bound traffic.
	_controller dispatcherController
)

type registerServiceFunc func(module *YARPCModule)

type yarpcConfig struct {
	modules.ModuleConfig
	Bind                    string
	AdvertiseName           string `yaml:"advertiseName"`
	inboundMiddleware       []middleware.UnaryInbound
	onewayInboundMiddleware []middleware.OnewayInbound
	inbounds                []transport.Inbound
}

// Stores a collection of all modules configs with a shared dispatcher
// that are safe to call from multiple go routines. All the configs must
// share the same AdvertiseName and represent a single service.
type dispatcherController struct {
	// sync configs
	sync.RWMutex

	// idempotent start and stop
	start      sync.Once
	stop       sync.Once
	stopError  error
	startError error

	configs    []*yarpcConfig
	dispatcher *yarpc.Dispatcher
}

// Adds the config to the collection, the first config sets up the AdvertiseName requirement on consequent additions.
func (c *dispatcherController) addConfig(config yarpcConfig) error {
	c.Lock()
	defer c.Unlock()

	if len(c.configs) > 0 && config.AdvertiseName != c.configs[0].AdvertiseName {
		return errs.Errorf("name mismatch, expected: %s; actual: %s",
			c.configs[0].AdvertiseName,
			config.AdvertiseName,
		)
	}

	c.configs = append(c.configs, &config)
	return nil
}

// Adds the default middleware: context propagation and auth.
func (c *dispatcherController) addDefaultMiddleware(host service.Host) error {
	cfg := yarpcConfig{
		AdvertiseName: host.Name(),
		inboundMiddleware: []middleware.UnaryInbound{
			contextInboundMiddleware{host},
			authInboundMiddleware{host},
		},
		onewayInboundMiddleware: []middleware.OnewayInbound{
			contextOnewayInboundMiddleware{host},
			authOnewayInboundMiddleware{host},
		},
	}

	if err := c.addConfig(cfg); err != nil {
		host.Logger().Error("Can't add the default middleware to configs", "error", err)
		return err
	}

	return nil
}

// Starts the dispatcher: wait until all modules call start, create a single dispatcher and then start it.
// Once started the collection will not start the dispatcher again.
func (c *dispatcherController) Start(host service.Host) error {
	c.start.Do(func() {
		var err error
		if err = c.addDefaultMiddleware(host); err != nil {
			c.startError = err
			return
		}

		var cfg yarpc.Config
		if cfg, err = c.mergeConfigs(); err != nil {
			c.startError = err
			return
		}

		_dispatcherMu.Lock()
		defer _dispatcherMu.Unlock()
		if c.dispatcher, err = _dispatcherFn(host, cfg); err != nil {
			c.startError = err
			return
		}

		c.startError = c.dispatcher.Start()
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
func (c *dispatcherController) mergeConfigs() (conf yarpc.Config, err error) {
	c.RLock()
	defer c.RUnlock()

	// Config collection should always have an additional config with the default middleware.
	if len(c.configs) <= 1 {
		return conf, errors.New("unable to merge empty configs")
	}

	conf.Name = c.configs[0].AdvertiseName

	// Collect all Inbounds and middleware from all configs
	var inboundMiddleware []middleware.UnaryInbound
	var onewayInboundMiddleware []middleware.OnewayInbound
	for _, cfg := range c.configs {
		conf.Inbounds = append(conf.Inbounds, cfg.inbounds...)
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
	reg registerServiceFunc,
	options ...modules.Option,
) (*YARPCModule, error) {
	name := "yarpc"
	if mi.Name != "" {
		name = mi.Name
	}

	module := &YARPCModule{
		ModuleBase: *modules.NewModuleBase(name, mi.Host, []string{}),
		register:   reg,
		config:     yarpcConfig{AdvertiseName: mi.Host.Name()},
	}

	stats.SetupRPCMetrics(mi.Host.Metrics())

	module.log = ulog.Logger().With("moduleName", name)
	for _, opt := range options {
		if err := opt(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", opt)
			return module, errs.Wrap(err, "unable to apply option to YARPC module")
		}
	}

	config := module.Host().Config().Scope("modules").Get(module.Name())
	if err := config.PopulateStruct(&module.config); err != nil {
		return module, err
	}

	module.config.inboundMiddleware = inboundMiddlewareFromCreateInfo(mi)
	module.config.onewayInboundMiddleware = onewayInboundMiddlewareFromCreateInfo(mi)

	tchTransport, err := tch.NewChannelTransport(
		tch.ServiceName(module.config.AdvertiseName),
		tch.ListenAddr(module.config.Bind),
	)

	if err != nil {
		return nil, err
	}

	// TODO(alsam): add support for the http transport
	module.config.inbounds = []transport.Inbound{tchTransport.NewInbound()}

	err = _controller.addConfig(module.config)
	return module, err
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

	if err := _controller.Start(m.Host()); err != nil {
		ret <- errs.Wrap(err, "unable to start dispatcher")
		return ret
	}

	m.register(m)
	m.log.Info("Module started",
		"name", m.config.AdvertiseName,
		"port", m.config.Bind)

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
	return _controller.Stop()
}

// IsRunning returns whether a module is running
func (m *YARPCModule) IsRunning() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.isRunning
}

type yarpcDispatcherFn func(service.Host, yarpc.Config) (*yarpc.Dispatcher, error)

// RegisterDispatcher allows you to override the YARPC dispatcher registration
func RegisterDispatcher(dispatchFn yarpcDispatcherFn) {
	_dispatcherMu.Lock()
	defer _dispatcherMu.Unlock()
	_dispatcherFn = dispatchFn
}

func defaultYARPCDispatcher(_ service.Host, cfg yarpc.Config) (*yarpc.Dispatcher, error) {
	return yarpc.NewDispatcher(cfg), nil
}

// Dispatcher returns a dispatcher that can be used to create clients.
// It should be called after at least one module have been started, otherwise it will be nil.
func Dispatcher() *yarpc.Dispatcher {
	return _controller.dispatcher
}
