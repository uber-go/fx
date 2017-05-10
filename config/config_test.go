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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
)

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
	t.Parallel()

	l := NewLoader()
	l.lookUp = func(string) (string, bool) {
		return "", false
	}

	l.SetConfigFiles("base", "development")
	l.SetEnvironmentPrefix("TEST")
	cfg := l.Load()

	assert.Equal(t, "global", cfg.Name())
	assert.Equal(t, "development", l.Environment())

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
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
	)

	v := provider.Get("n1.id1").WithDefault("xxx")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())
}

func TestScopedAccess(t *testing.T) {
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
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
	assert.NoError(t, v.Populate(nested))
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
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
	)

	str := &root{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	err := v.Populate(str)
	assert.Nil(t, err)

	assert.Equal(t, 1234, str.ID)
	assert.Equal(t, 1111, str.NestedPtr.ID1)
	assert.Equal(t, "2222", str.NestedPtr.ID2)
	assert.Equal(t, 111, str.Nested.ID1)
	assert.Equal(t, "aiden", str.Names[0])
	assert.Equal(t, "shawn", str.Names[1])
}

func TestArrayOfStructs(t *testing.T) {
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(structArrayYaml),
	)

	target := &arrayOfStructs{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	assert.NoError(t, v.Populate(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, 2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nest1),
	)
	target := &nested{}
	v := provider.Get(Root)
	assert.True(t, v.HasValue())
	assert.NoError(t, v.Populate(target))
	assert.Equal(t, "default_name", target.Name)
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
	t.Parallel()

	l := NewLoader()
	l.RegisterProviders(StaticProvider(map[string]interface{}{
		"hello": "world",
	}))

	l.RegisterDynamicProviders(func(dynamic Provider) (Provider, error) {
		return NewStaticProvider(map[string]interface{}{
			"dynamic": "provider",
		}), nil
	})

	cfg := l.Load()
	assert.Equal(t, "global", cfg.Name())
	assert.Equal(t, "world", cfg.Get("hello").AsString())
	assert.Equal(t, "provider", cfg.Get("dynamic").AsString())
	l.UnregisterProviders()
	assert.Nil(t, l.staticProviderFuncs)
	assert.Nil(t, l.dynamicProviderFuncs)
}

func TestNilProvider(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	l.RegisterProviders(func() (Provider, error) {
		return nil, errors.New("error creating Provider")
	})

	assert.Panics(t, func() { l.Load() }, "Can't initialize with nil provider")

	l.UnregisterProviders()
	l.RegisterProviders(func() (Provider, error) {
		return nil, nil
	})

	// don't panic on Load
	l.Load()

	l.UnregisterProviders()
	assert.Nil(t, l.staticProviderFuncs)
}

func expectedResolvePath(t *testing.T) string {
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	return path.Join(cwd, "testdata")
}

func TestResolvePath(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	res, err := l.ResolvePath("testdata")
	assert.NoError(t, err)
	assert.Equal(t, expectedResolvePath(t), res)
}

func TestResolvePathInvalid(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	res, err := l.ResolvePath("invalid")
	assert.Error(t, err)
	assert.Equal(t, "", res)
}

func TestResolvePathAbs(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	abs := expectedResolvePath(t)
	res, err := l.ResolvePath(abs)
	assert.NoError(t, err)
	assert.Equal(t, abs, res)
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

func TestPointerIntField(t *testing.T) {
	t.Parallel()

	type pointerFieldStruct struct {
		Name  string
		Value *int
	}

	var ptrYaml = `
ps:
  name: Hello
  value: 123
`
	p := NewYAMLProviderFromBytes([]byte(ptrYaml))

	cfg := &pointerFieldStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
}

func TestPointerTypedField(t *testing.T) {
	t.Parallel()

	type pointerFieldStruct struct {
		Name  string
		Value *int
	}

	var ptrPort = `
ps:
  name: Hello
  port: 123
`
	p := NewYAMLProviderFromBytes([]byte(ptrPort))

	cfg := &pointerFieldStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
}

func TestPointerChildTypedField(t *testing.T) {
	t.Parallel()

	type Port int
	type childPort struct {
		Port *Port
	}

	type portChildStruct struct {
		Name     string
		Child    *childPort
		Children []childPort
	}

	var ptrChildPort = `
ps:
  name: Hello
  child:
    port: 123
  children:
    - port: 321
`

	p := NewYAMLProviderFromBytes([]byte(ptrChildPort))

	cfg := &portChildStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
	require.Equal(t, 123, int(*cfg.Child.Port))
}

func TestRPCPortField(t *testing.T) {
	t.Parallel()

	type Port int
	type TChannelOutbound struct {
		Port *Port `yaml:"port"`
	}

	type Outbound struct {
		// Only one of the following must be set.
		TChannel *TChannelOutbound `yaml:"tchannel"`
	}

	type Outbounds []Outbound

	// Config is the YARPC YAML configuration.
	type YARPCConfig struct {
		// Name of the service.
		Name  string `yaml:"name"`
		Stuff int    `yaml:"stuff"`
		// Outbounds specifies how this service sends requests to other services.
		Outbounds Outbounds `yaml:"outbounds"`
	}

	var rpc = `
rpc:
  name: my-cool-service
  stuff: 999
  outbounds:
    - services:
        - buffetpushgateway
      tchannel:
        host: 127.0.0.1
        port: ${COMPANY_TCHANNEL_PORT:321}
`
	lookup := func(key string) (string) {
		if key == "COMPANY_TCHANNEL_PORT" {
			return "4324"
		}
		panic("boom")
	}

	p := NewExpandProvider(NewYAMLProviderFromReader(ioutil.NopCloser(bytes.NewBufferString(rpc))), lookup)

	cfg := &YARPCConfig{}
	v := p.Get("rpc")

	require.NoError(t, v.Populate(cfg))
	require.Equal(t, 4324, int(*cfg.Outbounds[0].TChannel.Port))
}

func TestLoader_Environment(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	l.SetLookupFn(func(key string) (string, bool) {
		require.Equal(t, "APP_ENVIRONMENT", key)
		return "KGBeast", true
	})

	assert.Equal(t, "KGBeast", l.Environment())
}

func TestLoader_AppRoot(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	l.SetLookupFn(func(key string) (string, bool) {
		require.Equal(t, "APP_ROOT", key)
		return "Harley Quinn", true
	})

	assert.Equal(t, "Harley Quinn", l.AppRoot())
}

func TestLoader_LoadPanicOnDynamicError(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	l.RegisterDynamicProviders(func(config Provider) (Provider, error) { return nil, errors.New("something scary") })

	assert.Panics(t, func() { l.Load() })
}

func withBase(t *testing.T, f func(dir string), contents string) {
	dir, err := ioutil.TempDir("", "TestLoader_Dirs")
	require.NoError(t, err)

	defer func() { require.NoError(t, os.Remove(dir)) }()

	base, err := os.Create(fmt.Sprintf("%s/base.yaml", dir))
	require.NoError(t, err)
	defer os.Remove(base.Name())

	base.WriteString(contents)
	base.Close()

	f(dir)
}

func TestLoader_Dirs(t *testing.T) {
	t.Parallel()

	f := func(dir string) {
		l := NewLoader()
		l.SetDirs(dir)
		p := l.Load()
		assert.Equal(t, "jocker", p.Get("vilain").String())
	}

	withBase(t, f, "vilain: jocker")
}

func TestParallelLoad(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	f := func(dir string) {
		l.SetDirs(dir)
		p := l.Load()
		assert.Equal(t, "bane", p.Get("vilain").String())
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	op := func() {
		withBase(t, f, "vilain: bane")
		wg.Done()
	}

	go op()
	go op()

	wg.Wait()
}

func TestZeroInitializeLoader(t *testing.T) {
	t.Parallel()
	var l Loader
	assert.NotPanics(t, func() { l.Load() })
}

func TestLoader_StaticProviderOrder(t *testing.T) {
	t.Parallel()
	f := func(dir string) {
		l := NewLoader(func() (Provider, error) {
			return NewStaticProvider(map[string]string{"value": "correct"}), nil
		})

		l.SetDirs(dir)
		p := l.Load()
		assert.Equal(t, "correct", p.Get("value").AsString())
	}

	withBase(t, f, "value: wrong")
}

func TestLoader_LoadFromCurrentFolder(t *testing.T) {
	t.Parallel()
	f := func(dir string) {
		l := NewLoader()
		l.SetConfigFiles(dir + "/base.yaml")
		p := l.Load()
		assert.Equal(t, "base", p.Get("value").AsString())
	}

	withBase(t, f, "value: base")
}

func TestLoader_LoadFromTestEnvironment(t *testing.T) {
	t.Parallel()
	f := func(dir string) {
		l := NewLoader()
		l.SetEnvironmentPrefix("MINI")
		f, err := os.Create(filepath.Join(dir, "spy.yaml"))
		require.NoError(t, err)
		defer os.Remove(f.Name())

		l.SetLookupFn(func(key string) (string, bool) {
			m := map[string]string{
				"MINI_CONFIG_DIR":  dir,
				"MINI_ENVIRONMENT": "spy",
			}

			res, ok := m[key]
			require.True(t, ok)
			return res, ok
		})

		f.WriteString("me: Austin Powers")
		p := l.Load()
		assert.Equal(t, "Austin Powers", p.Get("me").AsString())
		assert.Equal(t, "base", p.Get("value").AsString())
	}

	withBase(t, f, "value: base")
}
