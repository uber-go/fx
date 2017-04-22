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
	"reflect"

	"go.uber.org/dig"
	"go.uber.org/fx/config"
)

// Component is a dig constructor for something that's easily
// sharable in UberFx
type Component interface{}

// Module is a building block of UberFx
// TODO: Document and explain how is this different from Component?
// Something around roles and higher fidelity, maybe serving data
type Module interface {
	Name() string
	Constructor() Component
	Stop()
}

// Service foo
type Service struct {
	g                *dig.Graph
	modules          []Module
	moduleComponents []interface{}
	components       []Component
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

	// add a bunch of stuff
	for _, c := range modules {
		// TODO: everything is enabled right now
		co := c.Constructor()
		s.moduleComponents = append(s.moduleComponents, co)
		s.g.MustRegister(co)
	}

	return s
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
	// add a bunch of stuff
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, c := range s.moduleComponents {
		ctype := reflect.TypeOf(c)
		switch ctype.Kind() {
		case reflect.Func:
			objType := ctype.Out(0)
			s.g.MustResolve(reflect.New(objType).Interface())
		}
	}
}

// Stop foo
func (s *Service) Stop() {
	for _, m := range s.modules {
		m.Stop()
	}
}
