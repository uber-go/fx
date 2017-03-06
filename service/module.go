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

package service

import (
	"fmt"
	"sync"
)

// ModuleProvider provides Modules.
type ModuleProvider interface {
	// DefaultName is the name that will be used for a new Module
	// if no name is given as a ModuleOption.
	DefaultName() string
	// Create a new Module. The name of the Host and the scoping
	// of associated functions on the Host will be done using a name
	// provided by a ModuleOption, or by the DefaultName on this ModuleProvider.
	Create(Host) (Module, error)
}

// Module is the basic building block of an UberFx service.
type Module interface {
	// Start the Module. If an error is returned, the Module is assumed to be not started.
	// There is no need for this to be thread-safe, it will be called in a thread-safe manner.
	Start() error
	// Stop the Module. If an error is returned, the Module is still assumed to be stopped.
	// There is no need for this to be thread-safe, it will be called in a thread-safe manner.
	Stop() error
}

// ModuleProviderFromFunc creates a new ModuleProvider from a name and create function.
func ModuleProviderFromFunc(name string, createFunc func(Host) (Module, error)) ModuleProvider {
	return &moduleProviderFunc{name, createFunc}
}

// ModuleOption is a function that configures module creation.
type ModuleOption func(*moduleOptions) error

// WithModuleName will override the name given by the ModuleProvider.
func WithModuleName(name string) ModuleOption {
	return func(o *moduleOptions) error {
		o.name = name
		return nil
	}
}

// WithModuleRole will add a role to the Module.
//
// If the role was already added, this will be a no-op.
func WithModuleRole(role string) ModuleOption {
	return func(o *moduleOptions) error {
		if role == "" {
			return nil
		}
		for _, elem := range o.roles {
			if role == elem {
				return nil
			}
		}
		o.roles = append(o.roles, role)
		return nil
	}
}

// NewScopedHost returns a new Host scoped to a module. This should generally be used for testing.
func NewScopedHost(host Host, name string, roles ...string) (Host, error) {
	return newScopedHost(host, name, roles...)
}

type moduleProviderFunc struct {
	name       string
	createFunc func(Host) (Module, error)
}

func (m *moduleProviderFunc) DefaultName() string              { return m.name }
func (m *moduleProviderFunc) Create(host Host) (Module, error) { return m.createFunc(host) }

type moduleOptions struct {
	name  string
	roles []string
}

type moduleWrapper struct {
	module     Module
	scopedHost *scopedHost
	isRunning  bool
	lock       sync.RWMutex
}

func newModuleWrapper(
	host Host,
	moduleProvider ModuleProvider,
	options ...ModuleOption,
) (*moduleWrapper, error) {
	if moduleProvider == nil {
		return nil, nil
	}
	moduleOptions := &moduleOptions{}
	for _, option := range options {
		if err := option(moduleOptions); err != nil {
			return nil, err
		}
	}
	if moduleOptions.name == "" {
		moduleOptions.name = moduleProvider.DefaultName()
	}
	scopedHost, err := newScopedHost(
		host,
		moduleOptions.name,
		moduleOptions.roles...,
	)
	if err != nil {
		return nil, err
	}
	module, err := moduleProvider.Create(scopedHost)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, nil
	}
	return &moduleWrapper{module: module, scopedHost: scopedHost}, nil
}

func (m *moduleWrapper) Name() string {
	return m.scopedHost.name
}

func (m *moduleWrapper) Start() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.isRunning {
		return fmt.Errorf("module %s is running", m.Name())
	}
	if err := m.module.Start(); err != nil {
		return err
	}
	m.isRunning = true
	return nil
}

func (m *moduleWrapper) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.isRunning {
		return fmt.Errorf("module %s is not running", m.Name())
	}
	m.isRunning = false
	return m.module.Stop()
}

func (m *moduleWrapper) IsRunning() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.isRunning
}

type scopedHost struct {
	Host
	name  string
	roles []string
}

func newScopedHost(host Host, name string, roles ...string) (*scopedHost, error) {
	return &scopedHost{
		host,
		name,
		roles,
	}, nil
}

func (sh *scopedHost) Name() string {
	return sh.name
}

func (sh *scopedHost) Roles() []string {
	return sh.roles
}
