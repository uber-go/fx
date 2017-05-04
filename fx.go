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

	"go.uber.org/dig"
	"go.uber.org/zap"
)

// Component is a function that initializes and returns objects to be shared across the code
type Component interface{}

// Module is a building block or primary function performed by a service built on fx
type Module interface {
	Name() string
	Constructor() []Component
	Stop() error
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

	// setup metrics
	s.c.MustRegister(s.setupMetrics)

	s.c.MustRegister(setupRuntimeMetricsCollector)
	s.c.MustRegister(setupVersionMetricsEmitter)

	// Set up the logger
	s.c.MustRegister(s.setupLogger)

	s.c.MustRegister(s.setupTracer)

	// add a bunch of stuff
	for _, m := range modules {
		for _, ctor := range m.Constructor() {
			s.c.MustRegister(ctor)
		}
	}
	return s
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
			l.Info("Stopping ", zap.String("module", m.Name()))
			if err := m.Stop(); err != nil {
				l.Error("Error while stopping", zap.String("module", m.Name()), zap.Error(err))
			}

			l.Info("Stopped", zap.String("module", m.Name()))
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
