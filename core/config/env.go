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

func (p envConfigProvider) GetValue(key string, def interface{}) ConfigurationValue {
	env := toEnvString(p.prefix, key)

	var cv ConfigurationValue
	if value, found := p.provider.GetValue(env); found {
		cv = NewConfigurationValue(p, key, value, String, false, nil)
	} else {
		cv = NewConfigurationValue(p, key, def, String, true, nil)
	}
	return cv

}

func (p envConfigProvider) MustGetValue(key string) ConfigurationValue {
	return mustGetValue(p, key)
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
