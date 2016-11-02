package config

import "testing"

func BenchmarkSimpleGet(b *testing.B) {
	provider := NewYAMLProviderFromBytes([]byte(`foo: 1`))
	for n := 0; n < b.N; n++ {
		provider.GetValue("foo").AsInt()
	}
}
