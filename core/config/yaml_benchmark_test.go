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

import "testing"

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
	for n := 0; n < b.N; n++ {
		provider.GetValue("foo")
	}
}

func BenchmarkYAMLSimpleGetLevel3(b *testing.B) {
	provider := NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    baz: 1
`))
	for n := 0; n < b.N; n++ {
		provider.GetValue("foo.bar.baz")
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
	for n := 0; n < b.N; n++ {
		provider.GetValue("foo.bar.baz.alpha.bravo.charlie.foxtrot")
	}
}

func BenchmarkYAMLPopulateStruct(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	p := providerOneFile()

	for n := 0; n < b.N; n++ {
		c := &creds{}
		p.GetValue("api.credentials").PopulateStruct(c)
	}
}

func BenchmarkYAMLPopulateStructNested(b *testing.B) {
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

	for n := 0; n < b.N; n++ {
		s := &api{}
		p.GetValue("api").PopulateStruct(s)
	}
}

func BenchmarkYAMLPopulateStructNestedMultipleFiles(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	type api struct {
		URL         string
		Timeout     int
		Credentials creds
	}

	p := providerTwoFiles()

	for n := 0; n < b.N; n++ {
		s := &api{}
		p.GetValue("api").PopulateStruct(s)
	}
}

func providerOneFile() ConfigurationProvider {
	return NewYAMLProviderFromFiles(false, NewRelativeResolver("./testdata"), "benchmark1.yaml")
}

func providerTwoFiles() ConfigurationProvider {
	return NewYAMLProviderFromFiles(
		false,
		NewRelativeResolver("./testdata"),
		"benchmark1.yaml",
		"benchmark2.yaml",
	)
}
