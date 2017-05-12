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

	"github.com/uber-go/tally"
)

// ModuleProvider provides Modules.
type ModuleProvider interface {
	// DefaultName returns the default module name
	DefaultName() string
	// Create a new Module. The name of the Host and the scoping
	// of associated functions on the Host will be done using a name
	// provided by a ModuleOption, or by the DefaultName on this ModuleProvider.
	Create(tally.Scope) (Module, error)
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
func ModuleProviderFromFunc(
	name string,
	createFunc func(tally.Scope) (Module, error),
) ModuleProvider {
	return &moduleProvider{name: name, createFunc: createFunc}
}

// ModuleOption is a function that configures module creation.
type ModuleOption func(*moduleOptions) error

// WithName will override the root service name specified in config.
func WithName(name string) ModuleOption {
	return func(o *moduleOptions) error {
		o.ServiceName = name
		return nil
	}
}

// WithRole will add a role to the Module.
//
// If the role was already added, this will be a no-op.
func WithRole(role string) ModuleOption {
	return func(o *moduleOptions) error {
		if role == "" {
			return nil
		}
		for _, elem := range o.Roles {
			if role == elem {
				return nil
			}
		}
		o.Roles = append(o.Roles, role)
		return nil
	}
}

// WithModuleName will override the name given by the ModuleProvider.
func WithModuleName(name string) ModuleOption {
	return func(o *moduleOptions) error {
		o.ModuleName = name
		return nil
	}
}

// moduleOptions specifies options for service name and role
type moduleOptions struct {
	ModuleName  string
	ServiceName string
	Roles       []string
}

type moduleProvider struct {
	name       string
	createFunc func(tally.Scope) (Module, error)
}

func (m *moduleProvider) DefaultName() string                      { return m.name }
func (m *moduleProvider) Create(scope tally.Scope) (Module, error) { return m.createFunc(scope) }

type moduleWrapper struct {
	name        string
	serviceName string
	roles       []string
	module      Module
	isRunning   bool
	lock        sync.RWMutex
}

func newModuleWrapper(
	serviceName string,
	modRoles []string,
	scope tally.Scope,
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
	var svcName string
	var roles []string
	if moduleOptions.ModuleName == "" {
		moduleOptions.ModuleName = moduleProvider.DefaultName()
	}
	if moduleOptions.ServiceName != "" {
		svcName = moduleOptions.ServiceName
	} else {
		svcName = serviceName
	}
	if len(moduleOptions.Roles) > 0 {
		roles = moduleOptions.Roles
	} else {
		roles = modRoles
	}
	module, err := moduleProvider.Create(scope)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, nil
	}
	return &moduleWrapper{
		name:        moduleOptions.ModuleName,
		serviceName: svcName,
		module:      module,
		roles:       roles,
	}, nil
}

func (m *moduleWrapper) ServiceName() string {
	return m.serviceName
}

func (m *moduleWrapper) Name() string {
	return m.name
}

func (m *moduleWrapper) Roles() []string {
	return m.roles
}

func (m *moduleWrapper) Start() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.isRunning {
		return fmt.Errorf("module %s is running", m.name)
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
		return fmt.Errorf("module %s is not running", m.name)
	}
	m.isRunning = false
	return m.module.Stop()
}

func (m *moduleWrapper) IsRunning() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.isRunning
}
