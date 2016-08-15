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

type EnvironmentValueProvider interface {
	GetValue(key string) (string, bool)
}

var _ ConfigurationProvider = &envConfigProvider{}

// foo.bar -> [prefix]__foo__bar
func toEnvString(prefix string, key string) string {
	return fmt.Sprintf("%s__%s", prefix, strings.Replace(key, ".", "__", -1))
}

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

func (sp envConfigProvider) Scope(prefix string) ConfigurationProvider {
	return newScopedProvider(prefix, sp)
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
