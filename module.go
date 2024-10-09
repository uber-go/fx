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

	"go.uber.org/dig"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// A container represents a set of constructors to provide
// dependencies, and a set of functions to invoke once all the
// dependencies have been initialized.
//
// This definition corresponds to the dig.Container and dig.Scope.
type container interface {
	Invoke(interface{}, ...dig.InvokeOption) error
	Provide(interface{}, ...dig.ProvideOption) error
	Decorate(interface{}, ...dig.DecorateOption) error
}

// Module is a named group of zero or more fx.Options.
//
// A Module scopes the effect of certain operations to within the module.
// For more information, see [Decorate], [Replace], or [Invoke].
//
// Module allows packages to bundle sophisticated functionality into easy-to-use
// logical units.
// For example, a logging package might export a simple option like this:
//
//	package logging
//
//	var Module = fx.Module("logging",
//		fx.Provide(func() *log.Logger {
//			return log.New(os.Stdout, "", 0)
//		}),
//		// ...
//	)
//
// A shared all-in-one microservice package could use Module to bundle
// all required components of a microservice:
//
//	package server
//
//	var Module = fx.Module("server",
//		logging.Module,
//		metrics.Module,
//		tracing.Module,
//		rpc.Module,
//	)
//
// When new global functionality is added to the service ecosystem,
// it can be added to the shared module with minimal churn for users.
//
// Use the all-in-one pattern sparingly.
// It limits the flexibility available to the application.
func Module(name string, opts ...Option) Option {
	mo := moduleOption{
		name:     name,
		location: fxreflect.CallerStack(1, 2)[0],
		options:  opts,
	}
	return mo
}

type moduleOption struct {
	name     string
	location fxreflect.Frame
	options  []Option
}

func (o moduleOption) String() string {
	return fmt.Sprintf("fx.Module(%q, %v)", o.name, o.options)
}

func (o moduleOption) apply(mod *module) {
	// This get called on any submodules' that are declared
	// as part of another module.

	// 1. Create a new module with the parent being the specified
	// module.
	// 2. Apply child Options on the new module.
	// 3. Append it to the parent module.

	// Create trace as parent's trace with this module's location pre-pended.
	trace := append([]string{fmt.Sprintf("%v (%v)", o.location, o.name)}, mod.trace...)
	newModule := &module{
		name:   o.name,
		parent: mod,
		trace:  trace,
		app:    mod.app,
	}
	for _, opt := range o.options {
		opt.apply(newModule)
	}
	mod.modules = append(mod.modules, newModule)
}

type module struct {
	parent         *module
	name           string
	trace          []string
	scope          scope
	provides       []provide
	invokes        []invoke
	decorators     []decorator
	modules        []*module
	app            *App
	log            fxevent.Logger
	fallbackLogger fxevent.Logger
	logConstructor *provide
}

// scope is a private wrapper interface for dig.Container and dig.Scope.
// We can consider moving this into Fx using type constraints after Go 1.20
// is released and 1.17 is deprecated.
type scope interface {
	Decorate(f interface{}, opts ...dig.DecorateOption) error
	Invoke(f interface{}, opts ...dig.InvokeOption) error
	Provide(f interface{}, opts ...dig.ProvideOption) error
	Scope(name string, opts ...dig.ScopeOption) *dig.Scope
	String() string
}

// builds the Scopes using the App's Container. Note that this happens
// after applyModules' are called because the App's Container needs to
// be built for any Scopes to be initialized, and applys' should be called
// before the Container can get initialized.
func (m *module) build(app *App, root *dig.Container) {
	if m.parent == nil {
		m.scope = root
	} else {
		parentScope := m.parent.scope
		m.scope = parentScope.Scope(m.name)
		// use parent module's logger by default
		m.log = m.parent.log
	}

	if m.logConstructor != nil {
		// Since user supplied a custom logger, use a buffered logger
		// to hold all messages until user supplied logger is
		// instantiated. Then we flush those messages after fully
		// constructing the custom logger.
		m.fallbackLogger, m.log = m.log, new(logBuffer)
	}

	for _, mod := range m.modules {
		mod.build(app, root)
	}
}

func (m *module) provideAll() {
	for _, p := range m.provides {
		m.provide(p)
	}

	for _, m := range m.modules {
		m.provideAll()
	}
}

func (m *module) provide(p provide) {
	if m.app.err != nil {
		return
	}

	if p.IsSupply {
		m.supply(p)
		return
	}

	funcName := fxreflect.FuncName(p.Target)
	var info dig.ProvideInfo
	opts := []dig.ProvideOption{
		dig.FillProvideInfo(&info),
		dig.Export(!p.Private),
		dig.WithProviderCallback(func(ci dig.CallbackInfo) {
			m.log.LogEvent(&fxevent.Run{
				Name:       funcName,
				Kind:       "provide",
				ModuleName: m.name,
				Runtime:    ci.Runtime,
				Err:        ci.Error,
			})
		}),
	}

	if err := runProvide(m.scope, p, opts...); err != nil {
		m.app.err = err
	}
	outputNames := make([]string, len(info.Outputs))
	for i, o := range info.Outputs {
		outputNames[i] = o.String()
	}

	m.log.LogEvent(&fxevent.Provided{
		ConstructorName: funcName,
		StackTrace:      p.Stack.Strings(),
		ModuleTrace:     append([]string{p.Stack[0].String()}, m.trace...),
		ModuleName:      m.name,
		OutputTypeNames: outputNames,
		Err:             m.app.err,
		Private:         p.Private,
	})
}

func (m *module) supply(p provide) {
	typeName := p.SupplyType.String()
	opts := []dig.ProvideOption{
		dig.Export(!p.Private),
		dig.WithProviderCallback(func(ci dig.CallbackInfo) {
			m.log.LogEvent(&fxevent.Run{
				Name:       fmt.Sprintf("stub(%v)", typeName),
				Kind:       "supply",
				Runtime:    ci.Runtime,
				ModuleName: m.name,
			})
		}),
	}

	if err := runProvide(m.scope, p, opts...); err != nil {
		m.app.err = err
	}

	m.log.LogEvent(&fxevent.Supplied{
		TypeName:    typeName,
		StackTrace:  p.Stack.Strings(),
		ModuleTrace: append([]string{p.Stack[0].String()}, m.trace...),
		ModuleName:  m.name,
		Err:         m.app.err,
	})
}

// Constructs custom loggers for all modules in the tree
func (m *module) constructAllCustomLoggers() {
	if m.logConstructor != nil {
		if buffer, ok := m.log.(*logBuffer); ok {
			// default to parent's logger if custom logger constructor fails
			if err := m.constructCustomLogger(buffer); err != nil {
				m.app.err = multierr.Append(m.app.err, err)
				m.log = m.fallbackLogger
				buffer.Connect(m.log)
			}
		}
		m.fallbackLogger = nil
	} else if m.parent != nil {
		m.log = m.parent.log
	}

	for _, mod := range m.modules {
		mod.constructAllCustomLoggers()
	}
}

// Mirroring the behavior of app.constructCustomLogger
func (m *module) constructCustomLogger(buffer *logBuffer) (err error) {
	p := m.logConstructor
	fname := fxreflect.FuncName(p.Target)
	defer func() {
		m.log.LogEvent(&fxevent.LoggerInitialized{
			Err:             err,
			ConstructorName: fname,
		})
	}()

	// TODO: Use dig.FillProvideInfo to inspect the provided constructor
	// and fail the application if its signature didn't match.
	if err := m.scope.Provide(p.Target); err != nil {
		return fmt.Errorf("fx.WithLogger(%v) from:\n%+v\nin Module: %q\nFailed: %w",
			fname, p.Stack, m.name, err)
	}

	return m.scope.Invoke(func(log fxevent.Logger) {
		m.log = log
		buffer.Connect(log)
	})
}

func (m *module) executeInvokes() error {
	for _, m := range m.modules {
		if err := m.executeInvokes(); err != nil {
			return err
		}
	}

	for _, invoke := range m.invokes {
		if err := m.executeInvoke(invoke); err != nil {
			return err
		}
	}

	return nil
}

func (m *module) executeInvoke(i invoke) (err error) {
	fnName := fxreflect.FuncName(i.Target)
	m.log.LogEvent(&fxevent.Invoking{
		FunctionName: fnName,
		ModuleName:   m.name,
	})
	err = runInvoke(m.scope, i)
	m.log.LogEvent(&fxevent.Invoked{
		FunctionName: fnName,
		ModuleName:   m.name,
		Err:          err,
		Trace:        fmt.Sprintf("%+v", i.Stack), // format stack trace as multi-line
	})
	return err
}

func (m *module) decorateAll() error {
	for _, d := range m.decorators {
		if err := m.decorate(d); err != nil {
			return err
		}
	}

	for _, m := range m.modules {
		if err := m.decorateAll(); err != nil {
			return err
		}
	}
	return nil
}

func (m *module) decorate(d decorator) (err error) {
	if d.IsReplace {
		return m.replace(d)
	}

	funcName := fxreflect.FuncName(d.Target)
	var info dig.DecorateInfo
	opts := []dig.DecorateOption{
		dig.FillDecorateInfo(&info),
		dig.WithDecoratorCallback(func(ci dig.CallbackInfo) {
			m.log.LogEvent(&fxevent.Run{
				Name:       funcName,
				Kind:       "decorate",
				ModuleName: m.name,
				Runtime:    ci.Runtime,
				Err:        ci.Error,
			})
		}),
	}

	err = runDecorator(m.scope, d, opts...)
	outputNames := make([]string, len(info.Outputs))
	for i, o := range info.Outputs {
		outputNames[i] = o.String()
	}

	m.log.LogEvent(&fxevent.Decorated{
		DecoratorName:   funcName,
		StackTrace:      d.Stack.Strings(),
		ModuleTrace:     append([]string{d.Stack[0].String()}, m.trace...),
		ModuleName:      m.name,
		OutputTypeNames: outputNames,
		Err:             err,
	})

	return err
}

func (m *module) replace(d decorator) error {
	typeName := d.ReplaceType.String()
	opts := []dig.DecorateOption{
		dig.WithDecoratorCallback(func(ci dig.CallbackInfo) {
			m.log.LogEvent(&fxevent.Run{
				Name:       fmt.Sprintf("stub(%v)", typeName),
				Kind:       "replace",
				ModuleName: m.name,
				Runtime:    ci.Runtime,
				Err:        ci.Error,
			})
		}),
	}

	err := runDecorator(m.scope, d, opts...)
	m.log.LogEvent(&fxevent.Replaced{
		ModuleName:      m.name,
		StackTrace:      d.Stack.Strings(),
		ModuleTrace:     append([]string{d.Stack[0].String()}, m.trace...),
		OutputTypeNames: []string{typeName},
		Err:             err,
	})
	return err
}
