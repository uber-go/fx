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
	"bytes"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
)

type staticProvider struct {
	Provider
}

// NewStaticProvider should only be used in tests to isolate config from your environment
// It is not race free, because underlying objects can be accessed with Value().
func NewStaticProvider(data interface{}) Provider {
	return staticProvider{
		Provider: NewYAMLProviderFromReader(toReadCloser(data)),
	}
}

// NewStaticProviderWithExpand returns a static provider with values replaced by a mapping function.
func NewStaticProviderWithExpand(data interface{}, mapping func(string) (string, bool)) Provider {

	return staticProvider{
		Provider: NewYAMLProviderFromReaderWithExpand(mapping, toReadCloser(data)),
	}
}

// StaticProvider returns function to create StaticProvider during configuration initialization
func StaticProvider(data interface{}) ProviderFunc {
	return func() (Provider, error) {
		return NewStaticProvider(data), nil
	}
}

func (staticProvider) Name() string {
	return "static"
}

func toReadCloser(data interface{}) io.ReadCloser {
	b, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}

	return ioutil.NopCloser(bytes.NewBuffer(b))
}
