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

package config

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

// ScopedProvider defines recursive interface of providers based on the prefix
type ScopedProvider struct {
	Provider

	prefix string
}

// NewScopedProvider creates a child provider given a prefix
func NewScopedProvider(prefix string, provider Provider) *ScopedProvider {
	return &ScopedProvider{provider, prefix}
}

func addPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}

	if key == "" {
		return prefix
	}

	return prefix + "." + key
}

// Get returns configuration value
func (sp ScopedProvider) Get(key string) Value {
	return sp.Provider.Get(addPrefix(sp.prefix, key))
}

// Scope returns new scoped provider, given a prefix
func (sp ScopedProvider) Scope(prefix string) Provider {
	if prefix == "" {
		return sp
	}

	return NewScopedProvider(addPrefix(sp.prefix, prefix), sp.Provider)
}

// RegisterChangeCallback registers the callback in the underlying provider
func (sp ScopedProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
	return sp.Provider.RegisterChangeCallback(addPrefix(sp.prefix, key), callback)
}

// UnregisterChangeCallback un registers a callback in the underlying provider
func (sp ScopedProvider) UnregisterChangeCallback(key string) error {
	return sp.Provider.UnregisterChangeCallback(addPrefix(sp.prefix, key))
}
