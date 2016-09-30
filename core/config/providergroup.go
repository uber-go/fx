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

type ProviderGroup struct {
	name      string
	providers []ConfigurationProvider
}

func NewProviderGroup(name string, providers ...ConfigurationProvider) ConfigurationProvider {
	return ProviderGroup{
		name:      name,
		providers: providers,
	}
}

func (cc ProviderGroup) GetValue(key string) ConfigurationValue {
	cv := NewConfigurationValue(cc, key, nil, false, getValueType(nil), nil)

	// loop through the providers and return the value defined by the highest priority (e.g. last) provider
	for i := len(cc.providers) - 1; i >= 0; i-- {
		provider := cc.providers[i]
		if val := provider.GetValue(key); val.HasValue() && !val.IsDefault() {
			cv = val
			break
		}
	}

	// here we add a new root, which defines the "scope" at which
	// PopulateStructs will look for values.
	cv.root = cc
	return cv
}

func (p ProviderGroup) Name() string {
	return p.name
}

func (cc ProviderGroup) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) string {
	return ""
}
func (cc ProviderGroup) UnregisterChangeCallback(token string) bool {
	return false
}

func (cc ProviderGroup) Scope(prefix string) ConfigurationProvider {
	return newScopedProvider(prefix, cc)
}
