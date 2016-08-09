package config

import (
	"errors"
	"fmt"
)

type ConfigurationProvider interface {
	Name() string
	GetValue(key string, defaultValue interface{}) ConfigurationValue
	MustGetValue(key string) ConfigurationValue
}

type ConfigurationChangeCallback func(key string, provider string, configdata interface{})

type DynamicConfigurationProvider interface {
	ConfigurationProvider

	RegisterChangeCallback(key string, callback ConfigurationChangeCallback) string
	UnregisterChangeCallback(token string) bool
	Shutdown()
}

func keyNotFound(key string) error {
	return errors.New(fmt.Sprintf("Couldn't find key %q", key))
}

// helpers for common impls
func mustGetValue(p ConfigurationProvider, key string) ConfigurationValue {
	if val := p.GetValue(key, nil); !val.HasValue() {
		panic(keyNotFound(key))
	} else {
		return val
	}
}
