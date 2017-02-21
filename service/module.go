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
	"context"
	"fmt"
	"sync"

	"github.com/uber-go/tally"
	"go.uber.org/fx/ulog"
)

// Module is the basic building block of an UberFx service.
type Module interface {
	// Return the default name of the Module. This can be overridden with WithModuleName.
	Name() string
	// Start the Module. If an error is returned, the Module is assumed to be not started.
	// There is no need for this to be thread-safe, it will be called in a thread-safe manner.
	Start() error
	// Stop the Module. If an error is returned, the Module is still assumed to be stopped.
	// There is no need for this to be thread-safe, it will be called in a thread-safe manner.
	Stop() error
}

// ModuleInfo is the information passed to a Module on creation.
// This can be stored inside the module for use.
type ModuleInfo interface {
	Host
	Logger(context.Context) ulog.Log
	Items() map[string]interface{}
}

// NewModuleInfo returns a new ModuleInfo. This should generally be used for testing.
func NewModuleInfo(host Host, options ...ModuleOption) (ModuleInfo, error) {
	return newModuleInfo(host, options...)
}

// ModuleOption is a function that configures module creation.
type ModuleOption func(*moduleOptions) error

// WithModuleName will override the Module's default name.
//
// A WithModuleName option specifed later in the ModuleOptions list will
// override a WithName option earlier in the list.
func WithModuleName(name string) ModuleOption {
	return func(o *moduleOptions) error {
		o.name = name
		return nil
	}
}

// WithModuleRole will add a role to the Module.
//
// If the role was already added, this will be a no-op
func WithModuleRole(role string) ModuleOption {
	return func(o *moduleOptions) error {
		// TODO(pedge): is this desired? return an error?
		if role == "" {
			return nil
		}
		// TODO(pedge): is this desired? return an error?
		for _, elem := range o.roles {
			if role == elem {
				return nil
			}
		}
		o.roles = append(o.roles, role)
		return nil
	}
}

// WithModuleItem adds the value to the Module. If there is an existing value,
// it will be passed as the argument to the population function.
func WithModuleItem(key string, f func(interface{}) interface{}) ModuleOption {
	return func(o *moduleOptions) error {
		if value, ok := o.items[key]; ok {
			o.items[key] = f(value)
			return nil
		}
		o.items[key] = f(nil)
		return nil
	}
}

// ModuleCreateFunc handles instantiating modules from creation configuration,
type ModuleCreateFunc func(ModuleInfo) (Module, error)

type moduleOptions struct {
	name  string
	roles []string
	items map[string]interface{}
}

type moduleWrapper struct {
	module     Module
	moduleInfo *moduleInfo
	isRunning  bool
	lock       sync.RWMutex
}

func newModuleWrapper(host Host, moduleCreateFunc ModuleCreateFunc, options ...ModuleOption) (*moduleWrapper, error) {
	moduleInfo, err := newModuleInfo(host, options...)
	if err != nil {
		return nil, err
	}
	module, err := moduleCreateFunc(moduleInfo)
	if err != nil {
		return nil, err
	}
	if moduleInfo.name == "" {
		moduleInfo.name = module.Name()
	}
	return &moduleWrapper{module: module, moduleInfo: moduleInfo}, nil
}

func (m *moduleWrapper) Name() string {
	return m.moduleInfo.name
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

// TODO(pedge): we probably want to use service core to cache this stuff
// under the hood, as oppoed to making all these calls for scoping every time
type moduleInfo struct {
	Host
	name  string
	roles []string
	items map[string]interface{}
}

func newModuleInfo(host Host, options ...ModuleOption) (*moduleInfo, error) {
	moduleOptions := &moduleOptions{}
	for _, option := range options {
		if err := option(moduleOptions); err != nil {
			return nil, err
		}
	}
	return &moduleInfo{
		host,
		moduleOptions.name,
		moduleOptions.roles,
		moduleOptions.items,
	}, nil
}

// TODO(pedge): what about the Host's roles?
func (mi *moduleInfo) Roles() []string {
	return mi.roles
}

func (mi *moduleInfo) Metrics() tally.Scope {
	return mi.Host.Metrics().SubScope("modules").SubScope(mi.name)
}

func (mi *moduleInfo) Logger(ctx context.Context) ulog.Log {
	return ulog.Logger(ctx).With("module", mi.name)
}

// TODO(pedge): do we want to copy this?
// TODO(pedge): merge with concept of resources?
func (mi *moduleInfo) Items() map[string]interface{} {
	return mi.items
}
