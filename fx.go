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

package fx

import (
	"fmt"
	"reflect"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"

	"go.uber.org/dig"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"
)

// Component is a dig constructor for something that's easily
// sharable in UberFx
type Component interface{}

// Module is a building block of UberFx
// TODO: Document and explain how is this different from Component?
// Something around roles and higher fidelity, maybe serving data
type Module interface {
	Name() string
	Constructor(Core) []Component
	Stop()
}

// Service foo
type Service struct {
	g          *dig.Graph
	modules    []Module
	components []Component
	l          *zap.Logger
	core       Core
}

// Core has core
type Core struct {
	config config.Provider
	logger *zap.Logger
	scope  tally.Scope
	tracer opentracing.Tracer
}

// Logger returns the logger
func (c *Core) Logger() *zap.Logger {
	return c.logger
}

// Config returns the config
func (c *Core) Config() config.Provider {
	return c.config
}

// Metrics returns the metrics scope
func (c *Core) Metrics() tally.Scope {
	return c.scope
}

// Tracer returns the tracer
func (c *Core) Tracer() opentracing.Tracer {
	return c.tracer
}

// New foo
func New(modules ...Module) *Service {
	s := &Service{
		g:       dig.New(),
		modules: modules,
	}

	// load config for module creation and include it in the graph
	cfg := config.DefaultLoader.Load()
	// TODO: need to pull latest dig for direct Interface injection fix
	s.g.MustRegister(func() config.Provider { return cfg })

	// Set up the logger, remember it on the service and also inject into the graph
	l, err := logger(cfg)
	if err != nil {
		panic("failed to instantiate logger")
	}
	s.l = l
	s.g.MustRegister(l)

	s.core = Core{logger: l, config: cfg}
	// add a bunch of stuff
	for _, m := range modules {
		for _, ctor := range m.Constructor(s.core) {
			s.g.MustRegister(ctor)
		}
	}

	return s
}

func logger(cfg config.Provider) (*zap.Logger, error) {
	logConfig := ulog.Configuration{}
	err := logConfig.Configure(cfg.Get("logging"))
	if err != nil {
		return nil, err
	}
	l, err := logConfig.Build()
	return l, err
}

// WithComponents adds additional components to the service
func (s *Service) WithComponents(components ...Component) *Service {
	s.components = append(s.components, components...)

	// Add provided components to dig
	for _, c := range components {
		s.g.MustRegister(c)
	}

	return s
}

// Start foo
func (s *Service) Start() {
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, m := range s.modules {
		for _, ctor := range m.Constructor(s.core) {
			ctype := reflect.TypeOf(ctor)
			switch ctype.Kind() {
			case reflect.Func:
				objType := ctype.Out(0)
				fmt.Printf("Object %v %v\n", ctype, objType)
				s.g.MustResolve(reflect.New(objType).Interface())
			}
		}
		s.l.Info("Module started", zap.String("module", m.Name()))
	}

	s.l.Info("Service has started")
}

// Stop foo
func (s *Service) Stop() {
	for _, m := range s.modules {
		s.l.Info("Stopping module", zap.String("module", m.Name()))
		m.Stop()
		s.l.Info("Module stopped", zap.String("module", m.Name()))
	}
}
