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

type staticProvider struct {
	data map[string]interface{}
}

type scopedStaticProvider struct {
	Provider

	prefix string
}

// NewStaticProvider should only be used in tests to isolate config from your environment
func NewStaticProvider(data map[string]interface{}) Provider {
	return &staticProvider{
		data: data,
	}
}

// StaticProvider returns function to create StaticProvider during configuration initialization
func StaticProvider(data map[string]interface{}) ProviderFunc {
	return func() (Provider, error) {
		return NewStaticProvider(data), nil
	}
}

func (*staticProvider) Name() string {
	return "static"
}

func (s *staticProvider) Get(key string) Value {
	if key == "" {
		return NewValue(s, key, s.data, true, GetType(s.data), nil)
	}

	val, found := s.data[key]
	return NewValue(s, key, val, found, GetType(val), nil)
}

func (s *staticProvider) Scope(prefix string) Provider {
	return NewScopedProvider(prefix, s)
}

func (s *staticProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	// Static provider don't receive callback events
	return nil
}

func (s *staticProvider) UnregisterChangeCallback(token string) error {
	// Nothing to Unregister
	return nil
}

var _ Provider = &staticProvider{}
