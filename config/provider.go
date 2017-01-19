// Copyright (c) 2016 Uber Technologies, Inc.
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

package config

import "fmt"

// ConfigurationChangeCallback is called for updates of configuration data
type ConfigurationChangeCallback func(key string, provider string, configdata interface{})

// Root marks the root node in a Provider
const Root = ""

// A Provider provides a unified interface to accessing
// configuration systems.
type Provider interface {
	// the Name of the provider (YAML, Env, etc)
	Name() string
	// Get pulls a config value
	Get(key string) Value
	Scope(prefix string) Provider

	// A RegisterChangeCallback provides callback registration for config providers.
	// These callbacks are nop if a dynamic provider is not configured for the service.
	RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error
	UnregisterChangeCallback(token string) error
}

func keyNotFound(key string) error {
	return fmt.Errorf("couldn't find key %q", key)
}

// ScopedProvider defines recursive interface of providers based on the prefix
type ScopedProvider struct {
	Provider

	prefix string
}

// NewScopedProvider creates a child provider given a prefix
func NewScopedProvider(prefix string, provider Provider) Provider {
	return &ScopedProvider{provider, prefix}
}

// Get returns configuration value
func (sp ScopedProvider) Get(key string) Value {
	if sp.prefix != "" {
		key = sp.prefix + "." + key
	}

	return sp.Provider.Get(key)
}

// Scope returns new scoped provider, given a prefix
func (sp ScopedProvider) Scope(prefix string) Provider {
	return NewScopedProvider(prefix, sp)
}

// Register callback in the underling provider
func (sp ScopedProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
	if sp.prefix != "" {
		key = sp.prefix + "." + key
	}

	return sp.Provider.RegisterChangeCallback(key, callback)
}

// Unregister callback in the underling provider
func (sp ScopedProvider) UnregisterChangeCallback(key string) error {
	if sp.prefix != "" {
		key = sp.prefix + "." + key
	}

	return sp.Provider.UnregisterChangeCallback(key)
}
