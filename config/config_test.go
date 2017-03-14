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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var envValues = map[string]string{
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
appowner: owner@service.com
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
	assert.Equal(t, "development", Environment())

	cfg = NewProviderGroup("test", NewYAMLProviderFromBytes([]byte(`name: sample`)))
	assert.Equal(t, "test", cfg.Name())
}

func TestRootNodeConfig(t *testing.T) {
	t.Parallel()
	txt := []byte(`
one:
  two: hello
`)

	cfg := NewYAMLProviderFromBytes(txt).Get(Root).AsString()
	assert.Equal(t, "map[one:map[two:hello]]", cfg)
}

func TestDirectAccess(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)

	v := provider.Get("n1.id1").WithDefault("xxx")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())

	v2 := provider.Get("n1.id2").WithDefault("xxx")

	assert.True(t, v2.HasValue())
	assert.Equal(t, "-1", v2.Value())
}

func TestScopedAccess(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)

	p1 := provider.Get("n1")

	v1 := p1.Get("id1")
	v2 := p1.Get("idx").WithDefault("nope")

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
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)

	rpc := &rpcStruct{}
	v := provider.Get("modules.rpc")
	assert.True(t, v.HasValue())
	v.PopulateStruct(rpc)
	assert.Equal(t, ":8888", rpc.Bind)
}

func TestSimpleConfigValues(t *testing.T) {
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(yamlConfig3),
	)
	assert.Equal(t, 123, provider.Get("int").AsInt())
	assert.Equal(t, "test string", provider.Get("string").AsString())
	_, ok := provider.Get("nonexisting").TryAsString()
	assert.False(t, ok)
	assert.Equal(t, true, provider.Get("bool").AsBool())
	assert.Equal(t, 1.123, provider.Get("float").AsFloat())
	nested := &nested{}
	v := provider.Get("nonexisting")
	assert.NoError(t, v.PopulateStruct(nested))
}

func TestGetAsIntegerValue(t *testing.T) {
	t.Parallel()
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

func TestGetAsFloatValue(t *testing.T) {
	t.Parallel()
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
		cv := NewValue(NewStaticProvider(nil), "key", tc.value, true, Float, nil)
		assert.Equal(t, float64(2), cv.AsFloat())
	}
}

func TestNestedStructs(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)

	str := &root{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	err := v.PopulateStruct(str)
	assert.Nil(t, err)

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
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)

	target := &arrayOfStructs{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	assert.NoError(t, v.PopulateStruct(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, -2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nest1),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)
	target := &nested{}
	v := provider.Get(Root)
	assert.True(t, v.HasValue())
	assert.NoError(t, v.PopulateStruct(target))
	assert.Equal(t, "default_name", target.Name)
}

func TestDefaultValue(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: envValues}),
	)
	v := provider.Get("stuff")
	assert.False(t, v.HasValue())

	v = v.WithDefault("ok")

	assert.True(t, v.HasValue())
	assert.True(t, v.IsDefault())
	assert.Equal(t, "ok", v.Value())

	v2 := provider.Get("other_stuff")
	assert.False(t, v2.HasValue())
}

func TestInvalidConfigFailures(t *testing.T) {
	t.Parallel()
	valueType := []byte(`
id: xyz
boolean:
`)
	provider := NewYAMLProviderFromBytes(valueType)
	assert.Panics(t, func() { NewYAMLProviderFromBytes([]byte("bytes: \n\x010")) }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.Get("id").AsInt() }, "Can't parse as int")
	assert.Panics(t, func() { provider.Get("boolean").AsBool() }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.Get("id").AsFloat() }, "Can't parse as float")
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
	assert.Equal(t, "world", cfg.Get("hello").AsString())
	assert.Equal(t, "provider", cfg.Get("dynamic").AsString())
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

func TestGetConfigFiles(t *testing.T) {
	SetEnvironmentPrefix("TEST")

	files := getConfigFiles(baseFiles()...)
	assert.Contains(t, files, "./base.yaml")
	assert.Contains(t, files, "./development.yaml")
	assert.Contains(t, files, "./secrets.yaml")
	assert.Contains(t, files, "./config/base.yaml")
	assert.Contains(t, files, "./config/development.yaml")
	assert.Contains(t, files, "./config/secrets.yaml")
}

func TestSetConfigFiles(t *testing.T) {
	SetConfigFiles("x", "y")
	files := getConfigFiles(_configFiles...)
	assert.Contains(t, files, "./x.yaml")
	assert.Contains(t, files, "./y.yaml")
	assert.Contains(t, files, "./config/x.yaml")
	assert.Contains(t, files, "./config/y.yaml")
}

func expectedResolvePath(t *testing.T) string {
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	return path.Join(cwd, "testdata")
}

func TestResolvePath(t *testing.T) {
	res, err := ResolvePath("testdata")
	assert.NoError(t, err)
	assert.Equal(t, expectedResolvePath(t), res)
}

func TestResolvePathInvalid(t *testing.T) {
	res, err := ResolvePath("invalid")
	assert.Error(t, err)
	assert.Equal(t, "", res)
}

func TestResolvePathAbs(t *testing.T) {
	abs := expectedResolvePath(t)
	res, err := ResolvePath(abs)
	assert.NoError(t, err)
	assert.Equal(t, abs, res)
}

func TestEnvProviderWithEmptyPrefix(t *testing.T) {
	p := NewEnvProvider("", mapEnvironmentProvider{map[string]string{"key": "value"}})
	require.Equal(t, "value", p.Get("key").AsString())
	emptyScope := p.Get("")
	require.Equal(t, "value", emptyScope.Get("key").AsString())
	scope := emptyScope.Get("key")
	require.Equal(t, "value", scope.Get("").AsString())
}

func TestNopProvider_Get(t *testing.T) {
	t.Parallel()
	p := NopProvider{}
	assert.Equal(t, "NopProvider", p.Name())
	assert.NoError(t, p.RegisterChangeCallback("key", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))

	v := p.Get("randomKey")
	assert.Equal(t, "NopProvider", v.Source())
	assert.True(t, v.HasValue())
	assert.Nil(t, v.Value())
}
