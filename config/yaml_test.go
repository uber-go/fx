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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"go.uber.org/fx/testutils/env"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var yamlConfig1 = []byte(`
appid: keyvalue
desc: A simple keyvalue service
appowner: owner@service.com
modules:
  rpc:
    bind: :28941
`)

func TestYAMLSimple(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	c := provider.Get("modules.rpc.bind")
	assert.True(t, c.HasValue())
	assert.NotNil(t, c.Value())

	assert.Equal(t, ":28941", c.AsString())
}

func TestYAMLEnvInterpolation(t *testing.T) {
	t.Parallel()
	defer env.Override(t, "OWNER_EMAIL", "hello@there.yasss")()

	cfg := []byte(`
name: some name here
owner: ${OWNER_EMAIL}
module:
  fake:
    number: ${FAKE_NUMBER:321}`)

	p := NewYAMLProviderFromBytes(cfg)

	num, ok := p.Get("module.fake.number").TryAsFloat()
	require.True(t, ok)
	require.Equal(t, float64(321), num)

	owner := p.Get("owner").AsString()
	require.Equal(t, "hello@there.yasss", owner)
}

func TestYAMLEnvInterpolationMissing(t *testing.T) {
	t.Parallel()

	cfg := []byte(`
name: some name here
email: ${EMAIL_ADDRESS}`)

	require.Panics(t, func() {
		NewYAMLProviderFromBytes(cfg)
	})
}

func TestYAMLEnvInterpolationIncomplete(t *testing.T) {
	t.Parallel()

	cfg := []byte(`
name: some name here
telephone: ${SUPPORT_TEL:}`)

	require.Panics(t, func() {
		NewYAMLProviderFromBytes(cfg)
	})
}

func TestYAMLEnvInterpolationEmptyString(t *testing.T) {
	t.Parallel()

	cfg := []byte(`
name: ${APP_NAME:my shiny app}
fullTel: 1-800-LOLZ${TELEPHONE_EXTENSION:""}`)

	p := NewYAMLProviderFromBytes(cfg)
	require.Equal(t, "my shiny app", p.Get("name").AsString())
	require.Equal(t, "1-800-LOLZ", p.Get("fullTel").AsString())
}

type configStruct struct {
	AppID string
	Desc  string
	Owner string `yaml:"appowner"`
}

func TestYamlStructRoot(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := &configStruct{}

	assert.NoError(t, provider.Get(Root).PopulateStruct(cs))

	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "owner@service.com", cs.Owner)
}

type rpcStruct struct {
	Bind string `yaml:"bind"`
}

func TestYamlStructChild(t *testing.T) {
	t.Parallel()

	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := &rpcStruct{}

	assert.NoError(t, provider.Get("modules.rpc").PopulateStruct(cs))

	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromFiles(false, NewRelativeResolver("./testdata"), "base.yaml", "dev.yaml", "secrets.yaml")

	baseValue := provider.Get("value").AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.Get("value_override").AsString()
	assert.Equal(t, "dev_setting", devValue)

	secretValue := provider.Get("secret").AsString()
	assert.Equal(t, "my_secret", secretValue)
}

func TestAppRoot(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	assert.NoError(t, err)

	defer env.Override(t, _appRoot, path.Join(cwd, "testdata"))()
	provider := NewYAMLProviderFromFiles(false, NewRelativeResolver(), "base.yaml", "dev.yaml", "secrets.yaml")

	baseValue := provider.Get("value").AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.Get("value_override").AsString()
	assert.Equal(t, "dev_setting", devValue)

	secretValue := provider.Get("secret").AsString()
	assert.Equal(t, "my_secret", secretValue)
}

func TestNewYAMLProviderFromReader(t *testing.T) {
	t.Parallel()
	buff := bytes.NewBuffer([]byte(yamlConfig1))
	provider := NewYAMLProviderFromReader(ioutil.NopCloser(buff))
	cs := &configStruct{}
	assert.NoError(t, provider.Get(Root).PopulateStruct(cs))
	assert.Equal(t, "yaml", provider.Scope(Root).Name())
	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "owner@service.com", cs.Owner)
}

func TestYAMLNode(t *testing.T) {
	t.Parallel()
	buff := bytes.NewBuffer([]byte("a: b"))
	node := &yamlNode{value: make(map[interface{}]interface{})}
	err := unmarshalYAMLValue(ioutil.NopCloser(buff), &node.value)
	require.NoError(t, err)
	assert.Equal(t, "map[a:b]", node.String())
	assert.Equal(t, "map[interface {}]interface {}", node.Type().String())
}

func TestYamlNodeWithNil(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromFiles(false, nil)
	assert.NotNil(t, provider)
	assert.Panics(t, func() {
		_ = unmarshalYAMLValue(nil, nil)
	}, "Expected panic with nil inpout.")
}

func TestYamlNode_Callbacks(t *testing.T) {
	t.Parallel()
	p := NewYAMLProviderFromFiles(false, nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}

func withYamlBytes(yamlBytes []byte, f func(Provider)) {
	provider := NewProviderGroup("global", NewYAMLProviderFromBytes(yamlBytes))
	f(provider)
}

func TestMatchEmptyStruct(t *testing.T) {
	t.Parallel()
	withYamlBytes([]byte(``), func(provider Provider) {
		es := emptystruct{}
		provider.Get("emptystruct").PopulateStruct(&es)
		empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
		assert.True(t, reflect.DeepEqual(empty, es))
	})
}

func TestMatchPopulatedEmptyStruct(t *testing.T) {
	t.Parallel()
	withYamlBytes(emptyyaml, func(provider Provider) {
		es := emptystruct{}
		provider.Get("emptystruct").PopulateStruct(&es)
		empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
		assert.True(t, reflect.DeepEqual(empty, es))
	})
}

func TestPopulateStructWithPointers(t *testing.T) {
	t.Parallel()
	withYamlBytes(pointerYaml, func(provider Provider) {
		ps := pointerStruct{}
		provider.Get("pointerStruct").PopulateStruct(&ps)
		assert.True(t, *ps.MyTrueBool)
		assert.False(t, *ps.MyFalseBool)
		assert.Equal(t, "hello", *ps.MyString)
	})
}

func TestNonExistingPopulateStructWithPointers(t *testing.T) {
	t.Parallel()
	withYamlBytes([]byte(``), func(provider Provider) {
		ps := pointerStruct{}
		provider.Get("pointerStruct").PopulateStruct(&ps)
		assert.Nil(t, ps.MyTrueBool)
		assert.Nil(t, ps.MyFalseBool)
		assert.Nil(t, ps.MyString)
	})
}

func TestMapParsing(t *testing.T) {
	t.Parallel()
	withYamlBytes(complexMapYaml, func(provider Provider) {
		ms := mapStruct{}
		provider.Get("mapStruct").PopulateStruct(&ms)

		assert.NotNil(t, ms.MyMap)
		assert.NotZero(t, len(ms.MyMap))

		p, ok := ms.MyMap["policy"].(map[interface{}]interface{})
		assert.True(t, ok)

		for key, val := range p {
			assert.Equal(t, "makeway", key)
			assert.Equal(t, "notanoption", val)
		}

		assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
	})
}

func TestMapParsingSimpleMap(t *testing.T) {
	t.Parallel()
	withYamlBytes(simpleMapYaml, func(provider Provider) {
		ms := mapStruct{}
		provider.Get("mapStruct").PopulateStruct(&ms)
		assert.Equal(t, 1, ms.MyMap["one"])
		assert.Equal(t, 2, ms.MyMap["two"])
		assert.Equal(t, 3, ms.MyMap["three"])
		assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
	})
}

func TestMapParsingMapWithNonStringKeys(t *testing.T) {
	t.Parallel()
	withYamlBytes(intKeyMapYaml, func(provider Provider) {
		ik := intKeyMapStruct{}
		err := provider.Get("intKeyMapStruct").PopulateStruct(&ik)
		assert.NoError(t, err)
		assert.Equal(t, "onetwothree", ik.IntKeyMap[123])
	})
}

func TestDurationParsing(t *testing.T) {
	t.Parallel()
	withYamlBytes(durationYaml, func(provider Provider) {
		ds := durationStruct{}
		err := provider.Get("durationStruct").PopulateStruct(&ds)
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, ds.Seconds)
		assert.Equal(t, 20*time.Minute, ds.Minutes)
		assert.Equal(t, 30*time.Hour, ds.Hours)
	})
}

func TestParsingUnparsableDuration(t *testing.T) {
	t.Parallel()
	withYamlBytes(unparsableDurationYaml, func(provider Provider) {
		ds := durationStruct{}
		err := provider.Get("durationStruct").PopulateStruct(&ds)
		assert.Error(t, err)
	})
}

func TestTypeOfTypes(t *testing.T) {
	t.Parallel()
	withYamlBytes(typeStructYaml, func(provider Provider) {
		tts := typeStructStruct{}
		err := provider.Get(Root).PopulateStruct(&tts)
		assert.NoError(t, err)
		assert.Equal(t, userDefinedTypeInt(123), tts.TypeStruct.TestInt)
		assert.Equal(t, userDefinedTypeUInt(456), tts.TypeStruct.TestUInt)
		assert.Equal(t, userDefinedTypeFloat(123.456), tts.TypeStruct.TestFloat)
		assert.Equal(t, userDefinedTypeBool(true), tts.TypeStruct.TestBool)
		assert.Equal(t, userDefinedTypeString("hello"), tts.TypeStruct.TestString)
		assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
		assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
		assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
	})
}

func TestTypeOfTypesPtr(t *testing.T) {
	t.Parallel()
	withYamlBytes(typeStructYaml, func(provider Provider) {
		tts := typeStructStructPtr{}
		err := provider.Get(Root).PopulateStruct(&tts)
		assert.NoError(t, err)
		assert.Equal(t, userDefinedTypeInt(123), *tts.TypeStruct.TestInt)
		assert.Equal(t, userDefinedTypeUInt(456), *tts.TypeStruct.TestUInt)
		assert.Equal(t, userDefinedTypeFloat(123.456), *tts.TypeStruct.TestFloat)
		assert.Equal(t, userDefinedTypeBool(true), *tts.TypeStruct.TestBool)
		assert.Equal(t, userDefinedTypeString("hello"), *tts.TypeStruct.TestString)
		assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
		assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
		assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
	})
}

func TestTypeOfTypesPtrPtr(t *testing.T) {
	t.Parallel()
	withYamlBytes(typeStructYaml, func(provider Provider) {
		tts := typeStructStructPtrPtr{}
		err := provider.Get(Root).PopulateStruct(&tts)
		assert.NoError(t, err)
		assert.Equal(t, userDefinedTypeInt(123), *tts.TypeStruct.TestInt)
		assert.Equal(t, userDefinedTypeUInt(456), *tts.TypeStruct.TestUInt)
		assert.Equal(t, userDefinedTypeFloat(123.456), *tts.TypeStruct.TestFloat)
		assert.Equal(t, userDefinedTypeBool(true), *tts.TypeStruct.TestBool)
		assert.Equal(t, userDefinedTypeString("hello"), *tts.TypeStruct.TestString)
		assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
		assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
		assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
	})
}

func TestHappyTextUnMarshallerParsing(t *testing.T) {
	t.Parallel()
	withYamlBytes(happyTextUnmarshallerYaml, func(provider Provider) {
		ds := duckTales{}
		err := provider.Get("duckTales").PopulateStruct(&ds)
		assert.NoError(t, err)
		assert.Equal(t, scrooge, ds.Protagonist)
		assert.Equal(t, launchpadMcQuack, ds.Pilot)
	})
}

func TestGrumpyTextUnMarshallerParsing(t *testing.T) {
	t.Parallel()
	withYamlBytes(grumpyTextUnmarshallerYaml, func(provider Provider) {
		ds := duckTales{}
		err := provider.Get("darkwingDuck").PopulateStruct(&ds)
		assert.Contains(t, err.Error(), "Unknown character: DarkwingDuck")
	})
}

func TestMergeUnmarshaller(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromBytes(complexMapYaml, complexMapYamlV2)

	ms := mapStruct{}
	assert.NoError(t, provider.Get("mapStruct").PopulateStruct(&ms))
	assert.NotNil(t, ms.MyMap)
	assert.NotZero(t, len(ms.MyMap))

	p, ok := ms.MyMap["policy"].(map[interface{}]interface{})
	assert.True(t, ok)
	for key, val := range p {
		assert.Equal(t, "makeway", key)
		assert.Equal(t, "notanoption", val)
	}

	s, ok := ms.MyMap["pools"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, []interface{}{"very", "funny"}, s)
	assert.Equal(t, "", ms.NestedStruct.AdditionalData)
}

func TestMerge(t *testing.T) {
	t.Parallel()
	for _, v := range mergeTest {
		t.Run(v.description, func(t *testing.T) {
			prov := NewYAMLProviderFromBytes(v.yaml...)
			for path, exp := range v.expected {
				res := reflect.New(reflect.ValueOf(exp).Type()).Interface()
				assert.NoError(t, prov.Get(path).PopulateStruct(res))
				assert.Equal(t, exp, reflect.ValueOf(res).Elem().Interface(), "For path: %s", path)
			}
		})
	}
}

func TestMergePanics(t *testing.T) {
	t.Parallel()
	src := []byte(`
map:
  key: value
`)
	dst := []byte(`
map:
  - array
`)

	defer func() {
		if e := recover(); e != nil {
			assert.Contains(t, e, `can't merge map[interface{}]interface{} and []interface {}. Source: map["key":"value"]. Destination: ["array"]`)
			return
		}
		assert.Fail(t, "expected a panic")
	}()

	NewYAMLProviderFromBytes(dst, src)
}

func TestYamlProviderFmtPrintOnValueNoPanic(t *testing.T) {
	t.Parallel()
	provider := NewYAMLProviderFromBytes(yamlConfig1)
	c := provider.Get("modules.rpc.bind")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", c), "")
	}
	assert.NotPanics(t, f)
}

func TestArrayTypeNoPanic(t *testing.T) {
	t.Parallel()
	// This test will panic if we treat array the same as slice.
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := struct {
		ID [6]int `yaml:"id"`
	}{}

	assert.NoError(t, provider.Get(Root).PopulateStruct(&cs))
}

func TestNilYAMLProviderSetDefaultTagValue(t *testing.T) {
	t.Parallel()
	type Inner struct {
		Set bool `yaml:"set" default:"true"`
	}
	data := struct {
		ID0 int             `yaml:"id0" default:"10"`
		ID1 string          `yaml:"id1" default:"string"`
		ID2 Inner           `yaml:"id2"`
		ID3 []Inner         `yaml:"id3"`
		ID4 map[Inner]Inner `yaml:"id4"`
		ID5 *Inner          `yaml:"id5"`
		ID6 [6]Inner        `yaml:"id6"`
		ID7 [7]*Inner       `yaml:"id7"`
	}{}

	p := NewYAMLProviderFromBytes(nil)
	p.Get("hello").PopulateStruct(&data)

	assert.Equal(t, 10, data.ID0)
	assert.Equal(t, "string", data.ID1)
	assert.True(t, data.ID2.Set)
	assert.Nil(t, data.ID3)
	assert.Nil(t, data.ID4)
	assert.Nil(t, data.ID5)
	assert.True(t, data.ID6[0].Set)
	assert.Nil(t, data.ID7[0])
}

func TestDefaultWithMergeConfig(t *testing.T) {
	t.Parallel()
	base := []byte(`
abc:
  str: "base"
  int: 1
`)

	prod := []byte(`
abc:
  str: "prod"
`)
	cfg := struct {
		Str     string `yaml:"str" default:"nope"`
		Int     int    `yaml:"int" default:"0"`
		Bool    bool   `yaml:"bool" default:"true"`
		BoolPtr *bool  `yaml:"bool_ptr"`
	}{}
	p := NewYAMLProviderFromBytes(base, prod)
	p.Get("abc").PopulateStruct(&cfg)

	assert.Equal(t, "prod", cfg.Str)
	assert.Equal(t, 1, cfg.Int)
	assert.Equal(t, true, cfg.Bool)
	assert.Nil(t, cfg.BoolPtr)
}

func TestMapOfStructs(t *testing.T) {
	t.Parallel()
	type Bag struct {
		S string
		I int
		P *string
	}
	type Map struct {
		M map[string]Bag
	}

	b := []byte(`
m:
  first:
    s: one
    i: 1
  second:
    s: two
    i: 2
    p: Pointer
`)

	p := NewYAMLProviderFromBytes(b)
	var r Map
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, Bag{S: "one", I: 1, P: nil}, r.M["first"])

	snd := r.M["second"]
	assert.Equal(t, 2, snd.I)
	assert.Equal(t, "two", snd.S)
	assert.Equal(t, "Pointer", *snd.P)
}

func TestMapOfSlices(t *testing.T) {
	t.Parallel()
	type Map struct {
		S map[string][]time.Duration
	}

	b := []byte(`
s:
  first:
    - 1s
  second:
    - 2m
    - 3h
`)
	p := NewYAMLProviderFromBytes(b)
	var r Map
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, []time.Duration{time.Second}, r.S["first"])
	assert.Equal(t, []time.Duration{2 * time.Minute, 3 * time.Hour}, r.S["second"])
}

func TestMapOfArrays(t *testing.T) {
	t.Parallel()
	type Map struct {
		S map[string][2]time.Duration
	}

	b := []byte(`
s:
  first:
    - 1s
    - 4m
  second:
    - 2m
    - 3h
`)
	p := NewYAMLProviderFromBytes(b)
	var r Map
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, [2]time.Duration{time.Second, 4 * time.Minute}, r.S["first"])
	assert.Equal(t, [2]time.Duration{2 * time.Minute, 3 * time.Hour}, r.S["second"])
}

type cycle struct {
	A *cycle
}

type testProvider struct {
	staticProvider
	a cycle
}

func (s *testProvider) Get(key string) Value {
	val, found := s.a, true
	return NewValue(s, key, val, found, GetType(val), nil)
}

func TestLoops(t *testing.T) {
	t.Parallel()

	a := cycle{}
	a.A = &a

	b := cycle{&a}
	require.Equal(t, b, a)

	p := testProvider{}
	assert.Contains(t, p.Get(Root).PopulateStruct(&b).Error(), "cycles")
}

func TestInternalFieldsAreNotSet(t *testing.T) {
	t.Parallel()
	type External struct {
		internal string
	}

	b := []byte(`
internal: set
`)
	p := NewYAMLProviderFromBytes(b)
	var r External
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, "", r.internal)
}

func TestEmbeddedStructs(t *testing.T) {
	t.Skip("TODO(alsam) GFM(415)")
	t.Parallel()
	type Config struct {
		Foo string
	}

	type Sentry struct {
		DSN string
	}

	type Logging struct {
		Config
		Sentry
	}

	b := []byte(`
logging:
   foo: bar
   sentry:
      dsn: asdf
`)
	p := NewYAMLProviderFromBytes(b)
	var r Config
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, "bar", r.Foo)
}

func TestEmptyValuesSetForMaps(t *testing.T) {
	t.Parallel()
	type Hello interface {
		Hello()
	}

	type Foo struct {
		M map[string]Hello
	}

	b := []byte(`
M:
   sayHello:
`)
	p := NewYAMLProviderFromBytes(b)
	var r Foo
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Equal(t, r.M, map[string]Hello{"sayHello": Hello(nil)})
}

func TestEmptyValuesSetForStructs(t *testing.T) {
	t.Parallel()
	type Hello interface {
		Hello()
	}

	type Foo struct {
		Say Hello
	}

	b := []byte(`
Say:
`)
	p := NewYAMLProviderFromBytes(b)
	var r Foo
	require.NoError(t, p.Get(Root).PopulateStruct(&r))
	assert.Nil(t, r.Say)
}
