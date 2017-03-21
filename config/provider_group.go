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

type providerGroup struct {
	name      string
	providers []Provider
}

// NewProviderGroup creates a configuration provider from a group of backends
func NewProviderGroup(name string, providers ...Provider) Provider {
	group := providerGroup{
		name: name,
	}
	for _, provider := range providers {
		group.providers = append([]Provider{provider}, group.providers...)
	}
	return group
}

// WithProvider updates the current Provider
func (p providerGroup) WithProvider(provider Provider) Provider {
	return providerGroup{
		name:      p.name,
		providers: append([]Provider{provider}, p.providers...),
	}
}

func (p providerGroup) Get(key string) Value {
	cv := NewValue(p, key, nil, false, GetType(nil), nil)

	// loop through the providers and return the value defined by the highest priority provider
	for _, provider := range p.providers {
		if val := provider.Get(key); val.HasValue() && !val.IsDefault() {
			cv = val
			break
		}
	}

	// here we add a new root, which defines the "scope" at which
	// PopulateStructs will look for values.
	cv.root = p
	cv.provider = p
	return cv
}

func (p providerGroup) Name() string {
	return p.name
}

func (p providerGroup) RegisterChangeCallback(key string, callback ChangeCallback) error {
	for _, provider := range p.providers {
		if err := provider.RegisterChangeCallback(key, callback); err != nil {
			return err
		}
	}
	return nil
}

func (p providerGroup) UnregisterChangeCallback(token string) error {
	for _, provider := range p.providers {
		if err := provider.UnregisterChangeCallback(token); err != nil {
			return err
		}
	}
	return nil
}
