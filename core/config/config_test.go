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

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var env = map[string]string{
	toEnvString(defaultEnvPrefix, "modules.rpc.bind"): ":8888",
	toEnvString(defaultEnvPrefix, "n1.name"):          "struct_name",
	toEnvString(defaultEnvPrefix, "nptr.name"):        "ptr_name",
	toEnvString(defaultEnvPrefix, "nptr.id1"):         "999",
	toEnvString(defaultEnvPrefix, "n1.id2"):           "-1",
	toEnvString(defaultEnvPrefix, "names.0"):          "ai",
	toEnvString(defaultEnvPrefix, "things.2.id1"):     "-2",
}

type nested struct {
	Name string `yaml:"name" default:"default_name"`
	ID1  int    `yaml:"id1"`
	ID2  string `yaml:"id2"`
}

var nest1 = []byte(`
id1: 1
id2: 2
`)

type root struct {
	ID        int      `yaml:"id"`
	Names     []string `yaml:"names"`
	Nested    nested   `yaml:"n1"`
	NestedPtr *nested  `yaml:"nptr"`
}

var nestedYaml = []byte(`
id: 1234
names:
  - aiden
  - shawn
  - glib
  - madhu
  - anup
n1:
  name: struct
  id1:	111
  id2:  222
nptr:
  name: ptr
  id1: 	1111
  id2:  2222
`)

var structArrayYaml = []byte(`
things:
  - id1: 0
  - id1: 1
  - id1: 2
`)

var yamlConfig2 = []byte(`
appid: keyvalue
desc: A simple keyvalue service
appowner: uberfx@uber.com
modules:
  rpc:
    bind: :28941
`)

var yamlConfig3 = []byte(`
float: 1.123
bool: true
int: 123
string: test string
`)

type arrayOfStructs struct {
	Things []nested `yaml:"things"`
}

func TestGlobalConfig(t *testing.T) {
	SetEnvironmentPrefix("TEST")
	cfg := Load()

	assert.Equal(t, "global", cfg.Name())
	assert.Equal(t, "development", GetEnvironment())

	cfg = NewProviderGroup("test", NewYAMLProviderFromBytes([]byte(`applicationID: sample`)))
	assert.Equal(t, "test", cfg.Name())
}

func TestDirectAccess(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	v := provider.GetValue("n1.id1").WithDefault("xxx")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())

	v2 := provider.GetValue("n1.id2").WithDefault("xxx")

	assert.True(t, v2.HasValue())
	assert.Equal(t, "-1", v2.Value())
}

func TestScopedAccess(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	provider = provider.Scope("n1")

	v1 := provider.GetValue("id1")
	v2 := provider.GetValue("idx").WithDefault("nope")

	assert.True(t, v1.HasValue())
	assert.Equal(t, 111, v1.AsInt())
	assert.True(t, v2.IsDefault())
	assert.True(t, v2.HasValue())
	assert.Equal(t, v2.AsString(), "nope")
}

func TestOverrideSimple(t *testing.T) {

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(yamlConfig2),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	rpc := &rpcStruct{}
	v := provider.GetValue("modules.rpc")
	assert.True(t, v.HasValue())
	v.PopulateStruct(rpc)
	assert.Equal(t, ":8888", rpc.Bind)
}

func TestSimpleConfigValues(t *testing.T) {

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(yamlConfig3),
	)
	assert.Equal(t, 123, provider.GetValue("int").AsInt())
	assert.Equal(t, "test string", provider.GetValue("string").AsString())
	_, ok := provider.GetValue("nonexisting").TryAsString()
	assert.False(t, ok)
	assert.Equal(t, true, provider.GetValue("bool").AsBool())
	assert.Equal(t, 1.123, provider.GetValue("float").AsFloat())
	nested := &nested{}
	v := provider.GetValue("nonexisting")
	assert.NoError(t, v.PopulateStruct(nested))
}

func TestGetAsIntegerValue(t *testing.T) {
	testCases := []struct {
		value interface{}
	}{
		{float32(2)},
		{float64(2)},
		{int(2)},
		{int32(2)},
		{int64(2)},
	}
	for _, tc := range testCases {
		cv := NewValue(NewStaticProvider(nil), "key", tc.value, true, Integer, nil)
		assert.Equal(t, 2, cv.AsInt())
	}
}

func TestNestedStructs(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	str := &root{}

	v := provider.GetValue("")

	assert.True(t, v.HasValue())
	v.PopulateStruct(str)

	assert.Equal(t, 1234, str.ID)
	assert.Equal(t, 999, str.NestedPtr.ID1)
	assert.Equal(t, "2222", str.NestedPtr.ID2)
	assert.Equal(t, 111, str.Nested.ID1)
	assert.Equal(t, "-1", str.Nested.ID2)
	assert.Equal(t, "ai", str.Names[0])
	assert.Equal(t, "shawn", str.Names[1])
}

func TestArrayOfStructs(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(structArrayYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	target := &arrayOfStructs{}

	v := provider.GetValue("")

	assert.True(t, v.HasValue())
	assert.NoError(t, v.PopulateStruct(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, -2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nest1),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)
	target := &nested{}
	v := provider.GetValue("")
	assert.True(t, v.HasValue())
	assert.NoError(t, v.PopulateStruct(target))
	assert.Equal(t, "default_name", target.Name)
}

func TestDefaultValue(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)
	v := provider.GetValue("stuff")
	assert.False(t, v.HasValue())

	v = v.WithDefault("ok")

	assert.True(t, v.HasValue())
	assert.True(t, v.IsDefault())
	assert.Equal(t, "ok", v.Value())

	v2 := provider.GetValue("other_stuff")
	assert.False(t, v2.HasValue())
}

func TestInvalidConfigFailures(t *testing.T) {
	valueType := []byte(`
id: xyz
boolean:
`)
	provider := NewYAMLProviderFromBytes(valueType)
	assert.Panics(t, func() { NewYAMLProviderFromBytes([]byte("bytes: \n\x010")) }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.GetValue("id").AsInt() }, "Can't parse as int")
	assert.Panics(t, func() { provider.GetValue("boolean").AsBool() }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.GetValue("id").AsFloat() }, "Can't parse as float")
}

func TestRegisteredProvidersInitialization(t *testing.T) {
	RegisterProviders(StaticProvider(map[string]interface{}{
		"hello": "world",
	}))
	RegisterDynamicProviders(func(dynamic Provider) (Provider, error) {
		return NewStaticProvider(map[string]interface{}{
			"dynamic": "provider",
		}), nil
	})
	cfg := Load()
	assert.Equal(t, "global", cfg.Name())
	assert.Equal(t, "world", cfg.GetValue("hello").AsString())
	assert.Equal(t, "provider", cfg.GetValue("dynamic").AsString())
	UnregisterProviders()
	assert.Nil(t, _staticProviderFuncs)
	assert.Nil(t, _dynamicProviderFuncs)
}

func TestNilProvider(t *testing.T) {
	RegisterProviders(func() (Provider, error) {
		return nil, fmt.Errorf("error creating Provider")
	})
	assert.Panics(t, func() { Load() }, "Can't initialize with nil provider")

	oldProviders := _staticProviderFuncs
	defer func() {
		_staticProviderFuncs = oldProviders
	}()

	UnregisterProviders()
	RegisterProviders(func() (Provider, error) {
		return nil, nil
	})
	// don't panic on Load
	Load()

	UnregisterProviders()
	assert.Nil(t, _staticProviderFuncs)
}

func TestEnvProvider_Callbacks(t *testing.T) {
	p := NewEnvProvider("", nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}
