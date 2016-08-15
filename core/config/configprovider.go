package config

import (
	"errors"
	"fmt"
)

type ConfigurationProvider interface {
	Name() string // the name of the provider (YAML, Env, etc)
	GetValue(key string) ConfigurationValue
	Scope(prefix string) ConfigurationProvider
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

type scopedProvider struct {
	prefix string
	child  ConfigurationProvider
}

func newScopedProvider(prefix string, provider ConfigurationProvider) ConfigurationProvider {
	return &scopedProvider{prefix, provider}
}

func (sp scopedProvider) Name() string {
	return sp.child.Name()
}

func (sp scopedProvider) GetValue(key string) ConfigurationValue {
	if sp.prefix != "" {
		key = fmt.Sprintf("%s.%s", sp.prefix, key)
	}
	return sp.child.GetValue(key)
}

func (sp scopedProvider) Scope(prefix string) ConfigurationProvider {
	return newScopedProvider(prefix, sp)
}
