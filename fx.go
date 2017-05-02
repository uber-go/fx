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
	c              *dig.Container
	modules        []Module
	closers        []io.Closer
	loggerCloserFn func()
}

// New creates a service with the provided modules
func New(modules ...Module) *Service {
	s := &Service{
		c:       dig.New(),
		modules: modules,
	}

	// load config for module creation and include it in the graph
	s.c.MustRegister(config.DefaultLoader.Load)
	metrics.Freeze()

	s.c.MustRegister(s.setupMetrics)

	// Set up the logger, remember it on the service and also inject into the graph
	s.c.MustRegister(s.setupLogger)

	// add a bunch of stuff
	for _, m := range modules {
		for _, ctor := range m.Constructor() {
			s.c.MustRegister(ctor)
		}
	}
	return s
}

func (s *Service) setupMetrics(cfg config.Provider) (tally.Scope, tally.CachedStatsReporter) {
	scope, cachedStatsReporter, closer := metrics.RootScope(cfg)
	s.closers = append(s.closers, closer)
	return scope, cachedStatsReporter
}

func (s *Service) setupLogger(cp config.Provider, scope tally.Scope) (*zap.Logger, error) {
	logConfig := ulog.DefaultConfiguration()
	if cfg := cp.Get("logging"); cfg.HasValue() {
		if err := logConfig.Configure(cfg); err != nil {
			return nil, errors.Wrap(err, "failed to initialize logging from config")
		}
	}

	logger, err := logConfig.Build(zap.Hooks(ulog.Metrics(scope)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to build the logger")
	}
	s.loggerCloserFn = ulog.SetLogger(logger)
	return logger, err
}

// WithComponents adds additional components to the service
func (s *Service) WithComponents(components ...Component) *Service {
	// Add provided components to dig
	for _, c := range components {
		s.c.MustRegister(c)
	}
	return s
}

// Start starts the service
func (s *Service) Start() {
	if err := s.c.Invoke(func(l *zap.Logger) {
		for _, m := range s.modules {
			l.Info("Starting module", zap.String("module", m.Name()))
			for _, ctor := range m.Constructor() {
				if err := s.c.Invoke(ctor); err != nil {
					l.Error("error executing ctor ", zap.Error(err))
				}
			}
			l.Info("Module started", zap.String("module", m.Name()))
		}
		l.Info("Service has started")
	}); err != nil {
		zap.L().Error("Error starting the service", zap.Error(err))
	}
}

// Stop stops the service. Modules are stopped in reverse order to avoid errors caused by
// interdependencies
func (s *Service) Stop() {
	if err := s.c.Invoke(func(l *zap.Logger) {
		for i := len(s.modules) - 1; i >= 0; i-- {
			m := s.modules[i]
			l.Info("Stopping module", zap.String("module", m.Name()))
			m.Stop()
			l.Info("Module stopped", zap.String("module", m.Name()))
		}
		s.loggerCloserFn()
		for i := len(s.closers) - 1; i >= 0; i-- {
			if err := s.closers[i].Close(); err != nil {
				l.Error("Unable to close", zap.Error(err))
			}
		}
	}); err != nil {
		zap.L().Error("Error stopping the service", zap.Error(err))
	}
}
