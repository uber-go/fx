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
	// DefaultName returns the module name
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
	return &moduleProvider{name: name, createFunc: createFunc}
}

// ModuleOption is a function that configures module creation.
type ModuleOption func(*moduleOptions) error

// WithName will override the name given by the ModuleProvider.
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

// moduleOptions specifies options for service name and role
type moduleOptions struct {
	ServiceName string
	Roles       []string
}

// NewScopedHost returns a new Host scoped to a module. This should generally be used for testing.
func NewScopedHost(host Host, name string, roles ...string) (Host, error) {
	return newScopedHost(host, name, roles...), nil
}

type moduleProvider struct {
	name       string
	createFunc func(Host) (Module, error)
}

func (m *moduleProvider) DefaultName() string              { return m.name }
func (m *moduleProvider) Create(host Host) (Module, error) { return m.createFunc(host) }

type moduleWrapper struct {
	name       string
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
	var name string
	var roles []string
	if moduleOptions.ServiceName != "" {
		name = moduleOptions.ServiceName
	} else {
		name = host.Name()
	}
	if len(moduleOptions.Roles) > 0 {
		roles = moduleOptions.Roles
	} else {
		roles = host.Roles()
	}
	scopedHost := &scopedHost{Host: host, serviceName: name, roles: roles}
	module, err := moduleProvider.Create(scopedHost)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, nil
	}
	return &moduleWrapper{
		name:       moduleProvider.DefaultName(),
		module:     module,
		scopedHost: scopedHost,
	}, nil
}

func (m *moduleWrapper) Name() string {
	return m.name
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

// scopedHost is a host scoped to the module
type scopedHost struct {
	Host
	serviceName string
	roles       []string
	modName     string
}

func newScopedHost(host Host, serviceName string, roles ...string) *scopedHost {
	return &scopedHost{
		Host:        host,
		serviceName: serviceName,
		roles:       roles,
	}
}

// Name returns the scoped service name
func (sh *scopedHost) Name() string {
	return sh.serviceName
}

// Roles returns the roles for the module
func (sh *scopedHost) Roles() []string {
	return sh.roles
}
