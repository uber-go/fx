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

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
)

const (
	// ServiceNameKey is the config key of the service name
	ServiceNameKey = "name"
	// ServiceDescriptionKey is the config key of the service
	// description
	ServiceDescriptionKey = "description"
	// ServiceOwnerKey is the config key for a service owner
	ServiceOwnerKey = "owner"
)

const (
	_appRoot     = "APP_ROOT"
	_environment = "_ENVIRONMENT"
	_configDir   = "_CONFIG_DIR"
	_configRoot  = "./config"
	_baseFile    = "base"
	_secretsFile = "secrets"
)

var (
	_setupMux sync.Mutex

	_envPrefix            = "APP"
	_staticProviderFuncs  = []ProviderFunc{YamlProvider()}
	_configFiles          = baseFiles()
	_dynamicProviderFuncs []DynamicProviderFunc
)

var (
	_devEnv = "development"
)

// AppRoot returns the root directory of your application. UberFx developers
// can edit this via the APP_ROOT environment variable. If the environment
// variable is not set then it will fallback to the current working directory.
func AppRoot() string {
	if appRoot := os.Getenv(_appRoot); appRoot != "" {
		return appRoot
	}
	if cwd, err := os.Getwd(); err != nil {
		panic(fmt.Sprintf("Unable to get the current working directory: %s", err.Error()))
	} else {
		return cwd
	}
}

// ResolvePath returns an absolute path derived from AppRoot and the relative path.
// If the input parameter is already an absolute path it will be returned immediately.
func ResolvePath(relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return relative, nil
	}
	abs := path.Join(AppRoot(), relative)
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func getConfigFiles(fileSet ...string) []string {
	var files []string

	dirs := []string{".", _configRoot}
	for _, dir := range dirs {
		for _, baseFile := range fileSet {
			files = append(files, fmt.Sprintf("%s/%s.yaml", dir, baseFile))
		}
	}
	return files
}

func baseFiles() []string {
	env := Environment()
	return []string{_baseFile, env, _secretsFile}
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
		return NewYAMLProviderFromFiles(false, getResolver(), getConfigFiles(_configFiles...)...), nil
	}
}

// Environment returns current environment setup for the service
func Environment() string {
	env := os.Getenv(EnvironmentKey())
	if env == "" {
		env = _devEnv
	}
	return env
}

// Path returns path to the yaml configurations
func Path() string {
	configPath := os.Getenv(EnvironmentPrefix() + _configDir)
	if configPath == "" {
		configPath = _configRoot
	}
	return configPath
}

// SetConfigFiles overrides the set of available config files
// for the service
func SetConfigFiles(files ...string) {
	_configFiles = files
}

// SetEnvironmentPrefix sets environment prefix for the application
func SetEnvironmentPrefix(envPrefix string) {
	_envPrefix = envPrefix
}

// EnvironmentPrefix returns environment prefix for the application
func EnvironmentPrefix() string {
	return _envPrefix
}

// EnvironmentKey returns environment variable key name
func EnvironmentKey() string {
	return _envPrefix + _environment
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
