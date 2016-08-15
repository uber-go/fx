package config

import (
	"fmt"
	"os"
	"sync"
)

const (
	ApplicationIDKey          = "applicationid"
	ApplicationDescriptionKey = "applicationdesc"
	ApplicationOwnerKey       = "applicationowner"
)

var global ConfigurationProvider
var locked bool
var setupMux sync.Mutex

func Global() ConfigurationProvider {
	setupMux.Lock()
	defer setupMux.Unlock()
	locked = true
	return global
}

func ServiceName() string {
	return Global().GetValue(ApplicationIDKey).AsString()
}

func SetGlobal(provider ConfigurationProvider, force bool) {
	setupMux.Lock()
	defer setupMux.Unlock()
	if locked && !force {
		panic("Global provider must be set before any configuration access")
	}
	global = provider
}

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

	paths := []string{}

	configDir := os.Getenv("UBER_CONFIG_DIR")
	if configDir != "" {
		paths = []string{configDir}
	}

	resolver := NewRelativeResolver(paths...)

	// do the default thing
	global = NewProviderGroup("global", NewYamlProviderFromFiles(false, resolver, getUberConfigFiles()...), NewEnvProvider(defaultEnvPrefix, nil))
}
