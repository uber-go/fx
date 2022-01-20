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
)

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
	scope    *dig.Scope
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
	newModule := module{
		name:   o.Name,
		parent: mod,
	}

	for _, m := range o.modules {
		m.applyOnModule(&newModule)
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
	modules  []module
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

func (m *module) provide() {
	for _, provide := range m.provides {
		m.scope.Provide(provide.Target)
	}
}
