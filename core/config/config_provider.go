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

// A ConfigurationProvider provides a unified interface to accessing
// configuration systems.
type ConfigurationProvider interface {
	// the Name of the provider (YAML, Env, etc)
	Name() string
	// GetValue pulls a config value
	GetValue(key string) ConfigurationValue
	Scope(prefix string) ConfigurationProvider
}

// ConfigurationChangeCallback is called for updates of configuration data
type ConfigurationChangeCallback func(key string, provider string, configdata interface{})

// A DynamicConfigurationProvider provides configuration access as well as
// callback registration and shutdown hooks for dynamic config providers
type DynamicConfigurationProvider interface {
	ConfigurationProvider

	RegisterChangeCallback(key string, callback ConfigurationChangeCallback) string
	UnregisterChangeCallback(token string) bool
	Shutdown()
}

func keyNotFound(key string) error {
	return fmt.Errorf("couldn't find key %q", key)
}

type scopedProvider struct {
	ConfigurationProvider

	prefix string
}

func newScopedProvider(prefix string, provider ConfigurationProvider) ConfigurationProvider {
	return &scopedProvider{provider, prefix}
}

func (sp scopedProvider) GetValue(key string) ConfigurationValue {
	if sp.prefix != "" {
		key = fmt.Sprintf("%s.%s", sp.prefix, key)
	}
	return sp.ConfigurationProvider.GetValue(key)
}

func (sp scopedProvider) Scope(prefix string) ConfigurationProvider {
	return newScopedProvider(prefix, sp)
}
