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

package fx // import "go.uber.org/fx"

import (
	"io"
	"reflect"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/ulog"

	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

// Component is a function that initializes and returns objects to be shared across the code
type Component interface{}

// Module is a building block or primary function performed by a service built on fx
type Module interface {
	Name() string
	Constructor() []Component
	Stop()
}

// Service controls the lifecycle of an fx service
type Service struct {
	g              *dig.Graph
	modules        []Module
	scopeCloser    io.Closer
	logger         *zap.Logger
	loggerCloserFn func()
}

// scopeInit is used by the metrics package
// TODO: Remove after metrics refactor
type scopeInit struct {
	config config.Provider
}

// Name returns the name
func (c *scopeInit) Name() string {
	return c.config.Get(config.ServiceNameKey).AsString()
}

// Config returns the config
func (c *scopeInit) Config() config.Provider {
	return c.config
}

// New creates a service with the provided modules
func New(modules ...Module) *Service {
	s := &Service{
		g:       dig.New(),
		modules: modules,
	}

	// load config for module creation and include it in the graph
	cfg := config.DefaultLoader.Load()
	// TODO: need to pull latest dig for direct Interface injection fix
	s.g.MustRegister(func() config.Provider { return cfg })

	scope, _, scopeCloser := metrics.RootScope(&scopeInit{config: cfg})
	metrics.Freeze()
	s.g.MustRegister(func() (tally.Scope, error) { return scope, nil })
	s.scopeCloser = scopeCloser

	// Set up the logger, remember it on the service and also inject into the graph
	l, closerFn, err := createLogger(cfg, scope)
	if err != nil {
		panic("failed to instantiate logger")
	}
	s.logger = l
	s.loggerCloserFn = closerFn
	s.g.MustRegister(l)

	// add a bunch of stuff
	for _, m := range modules {
		for _, ctor := range m.Constructor() {
			s.g.MustRegister(ctor)
		}
	}

	return s
}

func createLogger(cp config.Provider, scope tally.Scope) (*zap.Logger, func(), error) {
	logConfig := ulog.DefaultConfiguration()
	if cfg := cp.Get("logging"); cfg.HasValue() {
		if err := logConfig.Configure(cfg); err != nil {
			return nil, nil, errors.Wrap(err, "failed to initialize logging from config")
		}
	}

	logger, err := logConfig.Build(zap.Hooks(ulog.Metrics(scope)))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to build the logger")
	}
	closerFn := ulog.SetLogger(logger)
	return logger, closerFn, err
}

// WithComponents adds additional components to the service
func (s *Service) WithComponents(components ...Component) *Service {
	// Add provided components to dig
	for _, c := range components {
		s.g.MustRegister(c)
	}
	return s
}

// Start starts the service
func (s *Service) Start() {
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, m := range s.modules {
		s.logger.Info("Starting module", zap.String("module", m.Name()))
		for _, ctor := range m.Constructor() {
			ctype := reflect.TypeOf(ctor)
			switch ctype.Kind() {
			case reflect.Func:
				objType := ctype.Out(0)
				s.g.MustResolve(reflect.New(objType).Interface())
			}
		}
		s.logger.Info("Module started", zap.String("module", m.Name()))
	}
	s.logger.Info("Service has started")
}

// Stop stops the service. Modules are stopped in reverse order to avoid errors caused by
// interdependencies
func (s *Service) Stop() {
	for i := len(s.modules) - 1; i >= 0; i-- {
		m := s.modules[i]
		s.logger.Info("Stopping module", zap.String("module", m.Name()))
		m.Stop()
		s.logger.Info("Module stopped", zap.String("module", m.Name()))
	}
	s.loggerCloserFn()
	if err := s.scopeCloser.Close(); err != nil {
		s.logger.Error("Unable to close scope", zap.Error(err))
	}
}
