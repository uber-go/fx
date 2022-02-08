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

// Module is a named group of zero or more fx.Options. A Module is a
// dependency graph with limited scope, and can be used for scoping
// graph modifications (not implemented yet).
func Module(name string, opts ...Option) Option {
	mo := moduleOption{
		name:    name,
		options: opts,
	}
	return mo
}

type moduleOption struct {
	name    string
	options []Option
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
	newModule := &module{
		name:   o.name,
		parent: mod,
		app:    mod.app,
	}
	for _, opt := range o.options {
		opt.apply(newModule)
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

// builds the Scopes using the App's Container. Note that this happens
// after applyModules' are called because the App's Container needs to
// be built for any Scopes to be initialized, and applys' should be called
// before the Container can get initialized.
func (m *module) build(app *App, root *dig.Container) {
	if m.parent == nil {
		m.scope = root.Scope(m.name)
	} else {
		parentScope := m.parent.scope
		m.scope = parentScope.Scope(m.name)
	}

	for i := 0; i < len(m.modules); i++ {
		m.modules[i].build(app, root)
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
