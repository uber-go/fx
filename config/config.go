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
	_configRoot  = "./config"
	_baseFile    = "base"
	_secretsFile = "secrets"
	_devEnv      = "development"
)

type lookUpFunc func(string) (string, bool)

// Loader is responsible for loading configs.
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

// DefaultLoader will be used if config is not defined for a service.
var DefaultLoader = newDefaultLoader()

func newDefaultLoader() *Loader {
	l := &Loader{
		envPrefix: "APP",
		dirs:      []string{".", "./config"},
		lookUp:    os.LookupEnv,
	}

	l.configFiles = l.baseFiles()
	l.RegisterProviders(l.YamlProvider())

	return l
}

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

func (l *Loader) joinFilepaths(fileSet ...string) []string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var files []string
	for _, dir := range l.dirs {
		for _, baseFile := range fileSet {
			files = append(files, filepath.Join(dir, fmt.Sprintf("%s.yaml", baseFile)))
		}
	}

	return files
}

func (l *Loader) baseFiles() []string {
	return []string{_baseFile, l.Environment(), _secretsFile}
}

func (l *Loader) getResolver() FileResolver {
	if dir := l.Path(); dir != "" {
		NewRelativeResolver(dir)
	}

	return NewRelativeResolver()
}

// YamlProvider returns function to create Yaml based configuration provider
func (l *Loader) YamlProvider() ProviderFunc {
	return func() (Provider, error) {
		return NewYAMLProviderFromFiles(false, l.getResolver(), l.joinFilepaths(l.getFiles()...)...), nil
	}
}

// Environment returns current environment setup for the service
func (l *Loader) Environment() string {
	if env, ok := l.lookUp(l.EnvironmentKey()); ok {
		return env
	}

	return _devEnv
}

// Path returns path to the yaml configurations
func (l *Loader) Path() string {
	if path, ok := l.lookUp(l.EnvironmentPrefix() + _configDir); ok {
		return path
	}

	return _configRoot
}

// SetConfigFiles overrides the set of available config files for the service.
func (l *Loader) SetConfigFiles(files ...string) {
	l.lock.Lock()

	l.configFiles = files

	l.lock.Unlock()
}

func (l *Loader) getFiles() []string {
	l.lock.RLock()
	l.lock.RUnlock()

	res := make([]string, len(l.configFiles))
	copy(res, l.configFiles)
	return res
}

// SetDirs overrides the set of dirs to load config files from.
func (l *Loader) SetDirs(dirs ...string) {
	l.lock.Lock()

	l.dirs = dirs

	l.lock.Unlock()
}

// SetEnvironmentPrefix sets environment prefix for the application.
func (l *Loader) SetEnvironmentPrefix(envPrefix string) {
	l.lock.Lock()

	l.envPrefix = envPrefix

	l.lock.Unlock()
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

	l.staticProviderFuncs = append(l.staticProviderFuncs, providerFuncs...)

	l.lock.Unlock()
}

// RegisterDynamicProviders registers dynamic config providers for the global config
// Dynamic provider initialization needs access to Provider for accessing necessary
// information for bootstrap, such as port number,keys, endpoints etc.
func (l *Loader) RegisterDynamicProviders(dynamicProviderFuncs ...DynamicProviderFunc) {
	l.lock.Lock()

	l.dynamicProviderFuncs = append(l.dynamicProviderFuncs, dynamicProviderFuncs...)

	l.lock.Unlock()
}

// UnregisterProviders clears all the default providers.
func (l *Loader) UnregisterProviders() {
	l.lock.Lock()

	l.staticProviderFuncs = nil
	l.dynamicProviderFuncs = nil

	l.lock.Unlock()
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
