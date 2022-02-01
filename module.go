// Copyright (c) 2022 Uber Technologies, Inc.
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

	"go.uber.org/dig"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxreflect"
)

// Module is a named group of zero or more fx.Options. A Module is a
// dependency graph with limited scope, and can be used for scoping
// graph modifications (not implemented yet).
func Module(name string, opts ...Option) Option {
	mo := moduleOption{
		Name: name,
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case provideOption:
			mo.provides = append(mo.provides, o)
		case invokeOption:
			mo.invokes = append(mo.invokes, o)
		case moduleOption:
			mo.modules = append(mo.modules, o)
		default:
			mo.options = append(mo.options, o)
		}
	}
	return mo
}

type moduleOption struct {
	Name     string
	modules  []moduleOption
	provides []provideOption
	invokes  []invokeOption
	options  []Option
}

func (o moduleOption) String() string {
	return fmt.Sprintf("module %s", o.Name)
}

func (o moduleOption) apply(app *App) {
	// This is the top-level module's apply.
	// Basically, this module acts as the "root" module that
	// connects all of its submodules to the App.
	// To do this, it needs to do the following, strictly in order:
	// 1. Create a new Scope.
	// 2. Create a new Module
	// 3. Apply any child modules on the new module
	// 4. Append the new module to the App's modules.

	newModule := &module{
		name: o.Name,
		app:  app,
	}

	for _, m := range o.modules {
		m.applyOnModule(newModule)
	}

	for _, p := range o.provides {
		newModule.provides = append(newModule.provides, p.getProvides()...)
	}

	for _, i := range o.invokes {
		newModule.invokes = append(newModule.invokes, i.getInvokes()...)
	}
	app.modules = append(app.modules, newModule)
}

func (o moduleOption) applyOnModule(mod *module) {
	newModule := &module{
		name:   o.Name,
		parent: mod,
		app:    mod.app,
	}

	for _, m := range o.modules {
		m.applyOnModule(newModule)
	}

	for _, p := range o.provides {
		newModule.provides = append(newModule.provides, p.getProvides()...)
	}

	for _, i := range o.invokes {
		newModule.invokes = append(newModule.invokes, i.getInvokes()...)
	}
	mod.modules = append(mod.modules, newModule)
}

type module struct {
	parent   *module
	name     string
	scope    *dig.Scope
	provides []provide
	invokes  []invoke
	modules  []*module
	app      *App
}

func (m *module) build(app *App) {
	// Builds the scopes using the info so far.

	if m.parent == nil {
		m.scope = app.container.Scope(m.name)
	} else {
		parentScope := m.parent.scope
		m.scope = parentScope.Scope(m.name)
	}

	for i := 0; i < len(m.modules); i++ {
		m.modules[i].build(app)
	}
}

func (m *module) provideAll() {
	for _, provide := range m.provides {
		m.provide(provide)
	}

	for _, m := range m.modules {
		m.provideAll()
	}
}

func (m *module) provide(p provide) {
	constructor := p.Target
	if _, ok := constructor.(Option); ok {
		m.app.err = fmt.Errorf("fx.Option should be passed to fx.New directly, "+
			"not to fx.Provide: fx.Provide received %v from:\n%+v",
			constructor, p.Stack)
		return
	}

	var info dig.ProvideInfo
	opts := []dig.ProvideOption{
		dig.FillProvideInfo(&info),
	}
	defer func() {
		var ev fxevent.Event

		switch {
		case p.IsSupply:
			ev = &fxevent.Supplied{
				TypeName: p.SupplyType.String(),
				Err:      m.app.err,
			}

		default:
			outputNames := make([]string, len(info.Outputs))
			for i, o := range info.Outputs {
				outputNames[i] = o.String()
			}

			ev = &fxevent.Provided{
				ConstructorName: fxreflect.FuncName(constructor),
				OutputTypeNames: outputNames,
				Err:             m.app.err,
			}
		}

		m.app.log.LogEvent(ev)
	}()

	switch constructor := constructor.(type) {
	case annotationError:
		// fx.Annotate failed. Turn it into an Fx error.
		m.app.err = fmt.Errorf(
			"encountered error while applying annotation using fx.Annotate to %s: %+v",
			fxreflect.FuncName(constructor.target), constructor.err)
		return

	case annotated:
		c, err := constructor.Build()
		if err != nil {
			m.app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", constructor, p.Stack, err)
			return
		}

		if err := m.scope.Provide(c, append(opts, dig.Export(true))...); err != nil {
			m.app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", constructor, p.Stack, err)
		}

	case Annotated:
		ann := constructor
		switch {
		case len(ann.Group) > 0 && len(ann.Name) > 0:
			m.app.err = fmt.Errorf(
				"fx.Annotated may specify only one of Name or Group: received %v from:\n%+v",
				ann, p.Stack)
			return
		case len(ann.Name) > 0:
			opts = append(opts, dig.Name(ann.Name))
		case len(ann.Group) > 0:
			opts = append(opts, dig.Group(ann.Group))
		}

		if err := m.scope.Provide(constructor, append(opts, dig.Export(true))...); err != nil {
			m.app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", ann, p.Stack, err)
		}

	default:
		if reflect.TypeOf(constructor).Kind() == reflect.Func {
			ft := reflect.ValueOf(constructor).Type()

			for i := 0; i < ft.NumOut(); i++ {
				t := ft.Out(i)

				if t == reflect.TypeOf(Annotated{}) {
					m.app.err = fmt.Errorf(
						"fx.Annotated should be passed to fx.Provide directly, "+
							"it should not be returned by the constructor: "+
							"fx.Provide received %v from:\n%+v",
						fxreflect.FuncName(constructor), p.Stack)
					return
				}
			}
		}

		if err := m.scope.Provide(constructor, append(opts, dig.Export(true))...); err != nil {
			m.app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", fxreflect.FuncName(constructor), p.Stack, err)
		}
	}
}

func (m *module) executeInvokes() error {
	for _, invoke := range m.invokes {
		if err := m.executeInvoke(invoke); err != nil {
			return err
		}
	}
	return nil
}

func (m *module) executeInvoke(i invoke) (err error) {
	fn := i.Target
	fnName := fxreflect.FuncName(fn)

	m.app.log.LogEvent(&fxevent.Invoking{FunctionName: fnName})
	defer func() {
		m.app.log.LogEvent(&fxevent.Invoked{
			FunctionName: fnName,
			Err:          err,
			Trace:        fmt.Sprintf("%+v", i.Stack), // format stack trace as multi-line
		})
	}()

	switch fn := fn.(type) {
	case Option:
		return fmt.Errorf("fx.Option should be passed to fx.New directly, "+
			"not to fx.Invoke: fx.Invoke received %v from:\n%+v",
			fn, i.Stack)

	case annotated:
		c, err := fn.Build()
		if err != nil {
			return err
		}

		return m.scope.Invoke(c)
	default:
		return m.scope.Invoke(fn)
	}
}
