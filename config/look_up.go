package config

import "os"

type lookupProvider struct {
	f func (key string)(interface{}, bool)
	*NopProvider
}

// EnvironmentProvider is a config provider to lookup for environment variables.
var EnvironmentProvider = NewLookupProvider(
	func(key string) (interface{}, bool){
		return os.LookupEnv(key)
	})

// NewLookupProvider returns a config provider that
func NewLookupProvider(lookup func (key string)(interface{}, bool)) Provider {
	return &lookupProvider{f: lookup}
}

func (l *lookupProvider) Get(key string) Value {
	if v, ok := l.f(key); ok {
		return NewValue(l, key, v, true, GetType(v), nil)
	}

	return NewValue(l, key, nil, false, Invalid, nil)
}

func (l *lookupProvider) Name() string {
	return "lookup"
}