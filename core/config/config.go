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
)

// TODO(ai) underscore-prefix these per Uber style
var (
	global   ConfigurationProvider
	locked   bool
	setupMux sync.Mutex
)

// Global returns the singleton configuration provider
func Global() ConfigurationProvider {
	setupMux.Lock()
	defer setupMux.Unlock()
	locked = true
	return global
}

// ServiceName returns the service's names
func ServiceName() string {
	return Global().GetValue(ApplicationIDKey).AsString()
}

// SetGlobal sets the singleton configuration provider
func SetGlobal(provider ConfigurationProvider, force bool) {
	setupMux.Lock()
	defer setupMux.Unlock()
	if locked && !force {
		panic("Global provider must be set before any configuration access")
	}
	global = provider
}

// ResetGlobal is used for tests
func ResetGlobal() {
	setupMux.Lock()
	defer setupMux.Unlock()
	global = nil
}

// TODO(ai) pull this out
// UBERSPECIFIC
func getUberConfigFiles() []string {

	env := os.Getenv("UBER_ENVIRONMENT")
	dc := os.Getenv("UBER_DATACENTER")

	var files []string
	if dc != "" && env != "" {
		files = append(files, fmt.Sprintf("./config/%s-%s.yaml", env, dc))
	}

	if env == "" {
		env = "development"
	}

	files = append(files, fmt.Sprintf("./config/%s.yaml", env), "./config/base.yaml")

	return files
}

func init() {
	// TODO(ai) see if we can do this without all the type assertions and long
	// lines
	paths := []string{}

	configDir := os.Getenv("UBER_CONFIG_DIR")
	if configDir != "" {
		paths = []string{configDir}
	}

	resolver := NewRelativeResolver(paths...)

	// do the default thing
	global = NewProviderGroup("global", NewYAMLProviderFromFiles(false, resolver, getUberConfigFiles()...), NewEnvProvider(defaultEnvPrefix, nil))
}
