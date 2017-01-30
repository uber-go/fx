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
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	errs "github.com/pkg/errors"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	tch "go.uber.org/yarpc/transport/tchannel"
)

// YARPCModule is an implementation of a core module using YARPC.
// All the YARPC modules share the same dispatcher and middleware.
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

	_ service.Module = &YARPCModule{}

	// Represents the global dispatcher used by all started modules.
	_dispatcher   *yarpc.Dispatcher
	_muDispatcher sync.RWMutex

	_configsOnce sync.Once
	_configs     configCollection
)

type registerServiceFunc func(module *YARPCModule)

type yarpcConfig struct {
	modules.ModuleConfig
	Bind                    string `yaml:"bind"`
	AdvertiseName           string `yaml:"advertiseName"`
	inboundMiddleware       []middleware.UnaryInbound
	onewayInboundMiddleware []middleware.OnewayInbound
	inbounds                []transport.Inbound
}

// Stores a collection of all modules configs and provides
// operations that are safe to call from multiple go routines.
type configCollection struct {
	sync.RWMutex
	configs []*yarpcConfig
}

func (c *configCollection) addConfig(config *yarpcConfig) error {
	c.Lock()
	defer c.Unlock()

	if len(c.configs) > 0 && config.AdvertiseName != c.configs[0].AdvertiseName {
		return errors.New("Name mismatch")
	}

	c.configs = append(c.configs, config)
	return nil
}

func (c *configCollection) removeConfig(config *yarpcConfig) error {
	c.Lock()
	defer c.Unlock()

	for i := range c.configs {
		if c.configs[i] == config {
			c.configs = append(c.configs[:i], c.configs[i+1:]...)
			return nil
		}
	}

	return errors.New("config not found")
}

func (c *configCollection) mergeConfigs() (conf yarpc.Config, err error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.configs) <= 1 {
		err = errors.New("empty configs")
		return
	}

	conf.Name = c.configs[0].AdvertiseName
	var inboundMiddleware []middleware.UnaryInbound
	var onewayInboundMiddleware []middleware.OnewayInbound
	for _, cfg := range c.configs {
		conf.Inbounds = append(conf.Inbounds, cfg.inbounds...)
		inboundMiddleware = append(inboundMiddleware, cfg.inboundMiddleware...)
		onewayInboundMiddleware = append(onewayInboundMiddleware, cfg.onewayInboundMiddleware...)
	}

	conf.InboundMiddleware = yarpc.InboundMiddleware{
		Unary:  yarpc.UnaryInboundMiddleware(inboundMiddleware...),
		Oneway: yarpc.OnewayInboundMiddleware(onewayInboundMiddleware...),
	}

	return conf, nil
}

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
		config: yarpcConfig{
			AdvertiseName: mi.Host.Name(),
		},
	}

	var err error
	_configsOnce.Do(func() {
		// All the modules are going to share the same middleware. This middleware is going to be the first:
		err = _configs.addConfig(&yarpcConfig{
			AdvertiseName: mi.Host.Name(),
			inboundMiddleware: []middleware.UnaryInbound{
				fxContextInboundMiddleware{mi.Host},
				authInboundMiddleware{mi.Host},
			},
			onewayInboundMiddleware: []middleware.OnewayInbound{
				fxContextOnewayInboundMiddleware{mi.Host},
				authOnewayInboundMiddleware{mi.Host},
			},
		})
	})

	if err != nil {
		module.log.Error("Unable to create a config for common middleware", "error", err)
		return module, errs.Wrap(err, "unable to create config with the common middleware for YARPC module")
	}

	module.log = ulog.Logger().With("moduleName", name)
	for _, opt := range options {
		if err := opt(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", opt)
			return module, errs.Wrap(err, "unable to apply option to YARPC module")
		}
	}

	err = module.Host().Config().Scope("modules").Get(module.Name()).PopulateStruct(&module.config)
	if err != nil {
		return module, err
	}

	module.config.inboundMiddleware = inboundMiddlewareFromCreateInfo(mi)
	module.config.onewayInboundMiddleware = onewayInboundMiddlewareFromCreateInfo(mi)

	tchTransport, err := tch.NewChannelTransport(
		tch.ServiceName(module.config.AdvertiseName),
		tch.ListenAddr(module.config.Bind),
	)

	module.config.inbounds = []transport.Inbound{tchTransport.NewInbound()}

	return module, err
}

// Start begins serving requests over RPC. It first stops the current dispatcher, merges configs from the global
// collection and then starts the new dispatcher.
func (m *YARPCModule) Start(readyCh chan<- struct{}) <-chan error {
	if m.IsRunning() {
		panic("module is already running")
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	ret := make(chan error, 1)
	if err := _configs.addConfig(&m.config); err != nil {
		ret <- errors.New("can't add module config " + err.Error())
		return ret
	}

	config, err := _configs.mergeConfigs()

	if err != nil {
		ret <- errors.New("unable to merge configs from multiple modules " + err.Error())
		return ret
	}

	dispatcher, err := _dispatcherFn(m.Host(), config)
	if err != nil {
		ret <- err
		return ret
	}

	if err := resetDispatcher(dispatcher); err != nil {
		ret <- err
		return ret
	}

	m.register(m)
	m.Host().Logger().Info("Module started",
		"name", m.config.AdvertiseName,
		"port", m.config.Bind)

	m.isRunning = true
	readyCh <- struct{}{}
	return ret
}

// Stop shuts down a YARPC module
func (m *YARPCModule) Stop() error {
	if !m.IsRunning() {
		panic("module is not running")
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	if err := _configs.removeConfig(&m.config); err != nil {
		return err
	}

	config, err := _configs.mergeConfigs()

	if err != nil {
		return errors.New("unable to merge configs from multiple modules " + err.Error())
	}

	dispatcher, err := _dispatcherFn(m.Host(), config)
	if err != nil {
		return err
	}

	if err := resetDispatcher(dispatcher); err != nil {
		return err
	}

	m.isRunning = false
	return nil
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
	_dispatcherFn = dispatchFn
}

func defaultYARPCDispatcher(_ service.Host, cfg yarpc.Config) (*yarpc.Dispatcher, error) {
	return yarpc.NewDispatcher(cfg), nil
}

// Dispatcher returns a dispatcher that can be used to create clients.
// It should be called after all modules have been started, because
// the each module start creates a new dispatcher and stops previous.
func Dispatcher() *yarpc.Dispatcher {
	_muDispatcher.RLock()
	defer _muDispatcher.RUnlock()
	return _dispatcher
}

// Stops the current dispatcher, sets the new one and starts it.
func resetDispatcher(d *yarpc.Dispatcher) error {
	_muDispatcher.Lock()
	defer _muDispatcher.Unlock()
	if _dispatcher == nil {
		_dispatcher = d
		return _dispatcher.Start()
	}

	if err := _dispatcher.Stop(); err != nil {
		return err
	}

	_dispatcher = d
	return _dispatcher.Start()
}
