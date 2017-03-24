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
	"testing"

	"go.uber.org/zap"
)

func BenchmarkYAMLCreateSingleFile(b *testing.B) {
	for n := 0; n < b.N; n++ {
		providerOneFile()
	}
}

func BenchmarkYAMLCreateMultiFile(b *testing.B) {
	for n := 0; n < b.N; n++ {
		providerTwoFiles()
	}
}

func BenchmarkYAMLSimpleGetLevel1(b *testing.B) {
	provider := NewYAMLProviderFromBytes([]byte(`foo: 1`))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo")
	}
}

func BenchmarkYAMLSimpleGetLevel3(b *testing.B) {
	provider := NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    baz: 1
`))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo.bar.baz")
	}
}

func BenchmarkYAMLSimpleGetLevel7(b *testing.B) {
	provider := NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    baz:
      alpha:
        bravo:
          charlie:
            foxtrot: 1
`))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo.bar.baz.alpha.bravo.charlie.foxtrot")
	}
}

func BenchmarkYAMLPopulate(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	p := providerOneFile()
	c := &creds{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api.credentials").Populate(c)
	}
}

func BenchmarkYAMLPopulateNested(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	type api struct {
		URL         string
		Timeout     int
		Credentials creds
	}

	p := providerOneFile()
	s := &api{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api").Populate(s)
	}
}

func BenchmarkYAMLPopulateNestedMultipleFiles(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	type api struct {
		URL         string
		Timeout     int
		Credentials creds
		Version     string
		Contact     string
	}

	p := providerTwoFiles()
	s := &api{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api").Populate(s)
	}
}

func BenchmarkYAMLPopulateNestedTextUnmarshaler(b *testing.B) {
	type protagonist struct {
		Hero duckTaleCharacter
	}

	type series struct {
		Episodes []protagonist
	}

	p := NewYAMLProviderFromFiles(true, NewRelativeResolver("./testdata"), "textUnmarshaller.yaml")
	s := &series{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get(Root).Populate(s)
	}
}

func BenchmarkZapConfigLoad(b *testing.B) {
	yaml := []byte(`
level: info
encoderConfig:
  levelEncoder: color
`)
	p := NewYAMLProviderFromBytes(yaml)
	cfg := &zap.Config{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		if err := p.Get(Root).Populate(cfg); err != nil {
			b.Error(err)
		}
	}
}

func providerOneFile() Provider {
	return NewYAMLProviderFromFiles(false, NewRelativeResolver("./testdata"), "benchmark1.yaml")
}

func providerTwoFiles() Provider {
	return NewYAMLProviderFromFiles(
		false,
		NewRelativeResolver("./testdata"),
		"benchmark1.yaml",
		"benchmark2.yaml",
	)
}
