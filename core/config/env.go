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

import (
	"fmt"
	"os"
	"strings"
)

type envConfigProvider struct {
	prefix   string
	provider EnvironmentValueProvider
}

const defaultEnvPrefix = "CONFIG"

// An EnvironmentValueProvider provides configuration from your environment
type EnvironmentValueProvider interface {
	GetValue(key string) (string, bool)
}

var _ ConfigurationProvider = &envConfigProvider{}

// foo.bar -> [prefix]__foo__bar
func toEnvString(prefix string, key string) string {
	return fmt.Sprintf("%s__%s", prefix, strings.Replace(key, ".", "__", -1))
}

// NewEnvProvider creates a configuration provider backed by an environment
func NewEnvProvider(prefix string, provider EnvironmentValueProvider) ConfigurationProvider {
	e := envConfigProvider{
		prefix:   prefix,
		provider: provider,
	}

	if provider == nil {
		e.provider = osEnvironmentProvider{}
	}
	return e
}

func (p envConfigProvider) Name() string {
	return "env"
}

func (p envConfigProvider) GetValue(key string) ConfigurationValue {
	env := toEnvString(p.prefix, key)

	var cv ConfigurationValue
	value, found := p.provider.GetValue(env)
	cv = NewConfigurationValue(p, key, value, found, String, nil)
	return cv
}

func (p envConfigProvider) Scope(prefix string) ConfigurationProvider {
	return NewScopedProvider(prefix, p)
}

func (p envConfigProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
	// Environments don't receive callback events
	return nil
}

func (p envConfigProvider) UnregisterChangeCallback(token string) error {
	// Nothing to Unregister
	return nil
}

type osEnvironmentProvider struct{}

func (p osEnvironmentProvider) GetValue(key string) (string, bool) {
	return os.LookupEnv(key)
}

type mapEnvironmentProvider struct {
	values map[string]string
}

func (p mapEnvironmentProvider) GetValue(key string) (string, bool) {
	val, ok := p.values[key]
	return val, ok
}
