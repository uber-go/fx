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

	m := os.Expand(fmt.Sprint(v.Value()), e.mapping)
	val = NewValue(e, key, m, true, String, &v.Timestamp)
	return
}

// RegisterChangeCallback registers the callback in the underlying provider.
func (e *expandProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	return e.p.RegisterChangeCallback(key, func(key string, provider string, data interface{}) {
		data = e.mapping(fmt.Sprint(data))
		callback(key, e.Name(), data)
	})
}

// UnregisterChangeCallback unregisters a callback in the underlying provider.
func (e *expandProvider) UnregisterChangeCallback(token string) error {
	return e.p.UnregisterChangeCallback(token)
}
