package config

import (
	"fmt"
	"os"
)

type expandProvider struct {
	p       Provider
	mapping func(string) string
}

// NewExpandProvider returns a config provider that uses a mapping function to expand ${var} or $var values in
// the values returned by the provider p.
func NewExpandProvider(p Provider, mapping func(string) string) Provider {
	return &expandProvider{p: p, mapping: mapping}
}

// Name returns expand
func (e *expandProvider) Name() string {
	return "expand"
}

// Get returns the value that has ${var} or $var replaced based on the mapping function.
func (e *expandProvider) Get(key string) (val Value) {
	v := e.p.Get(key)
	if !v.HasValue() {
		return NewValue(e, key, nil, false, Invalid, nil)
	}

	defer func() {
		recover()
		val = NewValue(e, key, val, true, v.Type, &v.Timestamp)
	}()

	m := os.Expand(fmt.Sprint(v.Value()), e.mapping)
	val = NewValue(e, key, m, true, String, &v.Timestamp)
	return
}

// RegisterChangeCallback registers the callback in the underlying provider.
func (e *expandProvider) RegisterChangeCallback(key string, callback ChangeCallback) error{
	return e.p.RegisterChangeCallback(key, func(key string, provider string, data interface{}){
		defer func() {
			recover()
			callback(key, e.Name(), data)
		}()

		data = e.mapping(fmt.Sprint(data))
	})
}

// UnregisterChangeCallback unregisters a callback in the underlying provider.
func (e *expandProvider) UnregisterChangeCallback(token string) error {
	return e.p.UnregisterChangeCallback(token)
}