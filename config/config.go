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

	flag "github.com/ogier/pflag"
)

const (
	// ServiceNameKey is the config key of the service name.
	ServiceNameKey = "name"

	// ServiceDescriptionKey is the config key of the service
	// description.
	ServiceDescriptionKey = "description"

	// ServiceOwnerKey is the config key for a service owner.
	ServiceOwnerKey = "owner"
)

const (
	_appRoot     = "_ROOT"
	_environment = "_ENVIRONMENT"
	_configDir   = "_CONFIG_DIR"
	_baseFile    = "base.yaml"
	_secretsFile = "secrets.yaml"
	_devEnv      = "development"
)

type lookUpFunc func(string) (string, bool)

// Loader is responsible for loading config providers.
type Loader struct {
	lock sync.RWMutex

	envPrefix            string
	staticProviderFuncs  []ProviderFunc
	dynamicProviderFuncs []DynamicProviderFunc

	// Files to load.
	configFiles []string

	// Dirs to load from.
	dirs []string

	// Where to look for environment variables.
	lookUp lookUpFunc
}

// DefaultLoader is going to be used by a service if config is not specified.
// First values are going to be looked in dynamic providers, then in command line provider
// and YAML provider is going to be the last.
var DefaultLoader = NewLoader(commandLineProviderFunc)

// NewLoader returns a default Loader with providers overriding the YAML provider.
func NewLoader(providers ...ProviderFunc) *Loader {
	l := &Loader{
		envPrefix: "APP",
		dirs:      []string{".", "./config"},
		lookUp:    os.LookupEnv,
	}

	// Order is important: we want users to be able to override static provider
	l.RegisterProviders(l.YamlProvider())
	l.RegisterProviders(providers...)

	return l
}

// TestConfig is Provider that can be used for testing. It loads configuration from
// base.yaml and test.yaml files.
var TestConfig = func() Provider {
	l := NewLoader()
	l.SetConfigFiles(_baseFile, "test.yaml")
	return l.Load()
}()

// AppRoot returns the root directory of your application. UberFx developers
// can edit this via the APP_ROOT environment variable. If the environment
// variable is not set then it will fallback to the current working directory.
func (l *Loader) AppRoot() string {
	if appRoot, ok := l.lookUp(l.EnvironmentPrefix() + _appRoot); ok {
		return appRoot
	}

	if cwd, err := os.Getwd(); err != nil {
		panic(fmt.Sprintf("Unable to get the current working directory: %q", err.Error()))
	} else {
		return cwd
	}
}

// ResolvePath returns an absolute path derived from AppRoot and the relative path.
// If the input parameter is already an absolute path it will be returned immediately.
func (l *Loader) ResolvePath(relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return relative, nil
	}

	abs := path.Join(l.AppRoot(), relative)
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}

	return abs, nil
}

func (l *Loader) baseFiles() []string {
	return []string{_baseFile, l.Environment() + ".yaml", _secretsFile}
}

func (l *Loader) getResolver() FileResolver {
	return NewRelativeResolver(l.Paths()...)
}

// YamlProvider returns function to create Yaml based configuration provider
func (l *Loader) YamlProvider() ProviderFunc {
	return func() (Provider, error) {
		return NewYAMLProviderFromFiles(false, l.getResolver(), l.getFiles()...), nil
	}
}

// Environment returns current environment setup for the service
func (l *Loader) Environment() string {
	if env, ok := l.lookUp(l.EnvironmentKey()); ok {
		return env
	}

	return _devEnv
}

// Paths returns paths to the yaml configurations
func (l *Loader) Paths() []string {
	if path, ok := l.lookUp(l.EnvironmentPrefix() + _configDir); ok {
		return []string{path}
	}

	return l.dirs
}

// SetConfigFiles overrides the set of available config files for the service.
func (l *Loader) SetConfigFiles(files ...string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.configFiles = files
}

func (l *Loader) getFiles() []string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	files := l.configFiles

	// Check if files where explicitly set.
	if len(files) == 0 {
		files = l.baseFiles()
	}

	res := make([]string, len(files))
	copy(res, files)
	return res
}

// SetDirs overrides the set of dirs to load config files from.
func (l *Loader) SetDirs(dirs ...string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.dirs = dirs
}

// SetEnvironmentPrefix sets environment prefix for the application.
func (l *Loader) SetEnvironmentPrefix(envPrefix string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.envPrefix = envPrefix
}

// EnvironmentPrefix returns environment prefix for the application.
func (l *Loader) EnvironmentPrefix() string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.envPrefix
}

// EnvironmentKey returns environment variable key name
func (l *Loader) EnvironmentKey() string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.envPrefix + _environment
}

// ProviderFunc is used to create config providers on configuration initialization.
type ProviderFunc func() (Provider, error)

// DynamicProviderFunc is used to create config providers on configuration initialization.
type DynamicProviderFunc func(config Provider) (Provider, error)

// RegisterProviders registers configuration providers for the global config.
func (l *Loader) RegisterProviders(providerFuncs ...ProviderFunc) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.staticProviderFuncs = append(l.staticProviderFuncs, providerFuncs...)
}

// RegisterDynamicProviders registers dynamic config providers for the global config
// Dynamic provider initialization needs access to Provider for accessing necessary
// information for bootstrap, such as port number,keys, endpoints etc.
func (l *Loader) RegisterDynamicProviders(dynamicProviderFuncs ...DynamicProviderFunc) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.dynamicProviderFuncs = append(l.dynamicProviderFuncs, dynamicProviderFuncs...)
}

// UnregisterProviders clears all the default providers.
func (l *Loader) UnregisterProviders() {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.staticProviderFuncs = nil
	l.dynamicProviderFuncs = nil
}

// Load creates a Provider for use in a service.
func (l *Loader) Load() Provider {
	var static []Provider
	for _, providerFunc := range l.staticProviderFuncs {
		cp, err := providerFunc()
		if err != nil {
			panic(err)
		}

		static = append(static, cp)
	}

	baseCfg := NewProviderGroup("global", static...)

	var dynamic []Provider
	for _, providerFunc := range l.dynamicProviderFuncs {
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

// SetLookupFn sets the lookup function to get environment variables.
func (l *Loader) SetLookupFn(fn func(string) (string, bool)) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.lookUp = fn
}

func commandLineProviderFunc() (Provider, error) {
	var s StringSlice
	flag.CommandLine.Var(&s, "roles", "")
	return NewCommandLineProvider(flag.CommandLine, os.Args[1:]), nil
}
