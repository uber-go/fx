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
	"sync"
)

const (
	// ApplicationIDKey is the identifier of an application ID
	ApplicationIDKey = "applicationID"
	// ApplicationDescriptionKey is the configuration key of the application's
	// description
	ApplicationDescriptionKey = "applicationDesc"
	// ApplicationOwnerKey is the configuration key for an application's owner
	ApplicationOwnerKey = "applicationOwner"

	environment = "_ENVIRONMENT"
	datacenter  = "_DATACENTER"
	configdir   = "_CONFIG_DIR"
	config      = "config"
)

var (
	_setupMux sync.Mutex

	_envPrefix            = "APP"
	_staticProviderFuncs  = []ProviderFunc{YamlProvider(), EnvProvider()}
	_dynamicProviderFuncs []DynamicProviderFunc
)

func getConfigFiles() []string {
	env := GetEnvironment()
	dc := os.Getenv(GetEnvironmentPrefix() + datacenter)

	var files []string
	if dc != "" && env != "" {
		files = append(files, fmt.Sprintf("./%s/%s-%s.yaml", config, env, dc))
	}
	files = append(files,
		fmt.Sprintf("./%s/%s.yaml", config, env),
		fmt.Sprintf("./%s/base.yaml", config))

	return files
}

func getResolver() FileResolver {
	paths := []string{}
	configDir := Path()
	if configDir != "" {
		paths = []string{configDir}
	}
	return NewRelativeResolver(paths...)
}

// YamlProvider returns function to create Yaml based configuration provider
func YamlProvider() ProviderFunc {
	return func() (Provider, error) {
		return NewYAMLProviderFromFiles(false, getResolver(), getConfigFiles()...), nil
	}
}

// EnvProvider returns function to create environment based config provider
func EnvProvider() ProviderFunc {
	return func() (Provider, error) {
		return NewEnvProvider(defaultEnvPrefix, nil), nil
	}
}

// GetEnvironment returns current environment setup for the service
func GetEnvironment() string {
	env := os.Getenv(GetEnvironmentPrefix() + environment)
	if env == "" {
		env = "development"
	}
	return env
}

// Path returns path to the yaml configurations
func Path() string {
	configPath := os.Getenv(GetEnvironmentPrefix() + configdir)
	if configPath == "" {
		configPath = config
	}
	return configPath
}

// SetEnvironmentPrefix sets environment prefix for the application
func SetEnvironmentPrefix(envPrefix string) {
	_envPrefix = envPrefix
}

// GetEnvironmentPrefix returns environment prefix for the application
func GetEnvironmentPrefix() string {
	return _envPrefix
}

// ProviderFunc is used to create config providers on configuration initialization
type ProviderFunc func() (Provider, error)

// DynamicProviderFunc is used to create config providers on configuration initialization
type DynamicProviderFunc func(config Provider) (Provider, error)

// RegisterProviders registers configuration providers for the global config
func RegisterProviders(providerFuncs ...ProviderFunc) {
	_setupMux.Lock()
	defer _setupMux.Unlock()
	_staticProviderFuncs = append(_staticProviderFuncs, providerFuncs...)
}

// RegisterDynamicProviders registers dynamic config providers for the global config
// Dynamic provider initialization needs access to Provider for accessing necessary
// information for bootstrap, such as port number,keys, endpoints etc.
func RegisterDynamicProviders(dynamicProviderFuncs ...DynamicProviderFunc) {
	_setupMux.Lock()
	defer _setupMux.Unlock()
	_dynamicProviderFuncs = append(_dynamicProviderFuncs, dynamicProviderFuncs...)
}

// Providers should only be used during tests
func Providers() []ProviderFunc {
	return _staticProviderFuncs
}

// UnregisterProviders clears all the default providers
func UnregisterProviders() {
	_setupMux.Lock()
	defer _setupMux.Unlock()
	_staticProviderFuncs = nil
	_dynamicProviderFuncs = nil
}

// Load creates a Provider for use in a service
func Load() Provider {
	var static []Provider
	for _, providerFunc := range _staticProviderFuncs {
		cp, err := providerFunc()
		if err != nil {
			panic(err)
		}
		static = append(static, cp)
	}
	baseCfg := NewProviderGroup("global", static...)

	var dynamic = make([]Provider, 0, 2)
	for _, providerFunc := range _dynamicProviderFuncs {
		cp, err := providerFunc(baseCfg)
		if err != nil {
			panic(err)
		}
		if cp != nil {
			dynamic = append(dynamic, cp)
		}
	}
	return NewProviderGroup("global", append(static, dynamic...)...)
}
