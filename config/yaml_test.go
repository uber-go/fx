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

func TestYamlSimple(t *testing.T) {
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	c := provider.Get("modules.rpc.bind")
	assert.True(t, c.HasValue())
	assert.NotNil(t, c.Value())

	assert.Equal(t, ":28941", c.AsString())
}

type configStruct struct {
	AppID string
	Desc  string
	Owner string `yaml:"appowner"`
}

func TestYamlStructRoot(t *testing.T) {
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
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := &rpcStruct{}

	assert.NoError(t, provider.Get("modules.rpc").PopulateStruct(cs))

	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	provider := NewYAMLProviderFromFiles(false, NewRelativeResolver("./testdata"), "base.yaml", "dev.yaml", "secrets.yaml")

	baseValue := provider.Get("value").AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.Get("value_override").AsString()
	assert.Equal(t, "dev_setting", devValue)

	secretValue := provider.Get("secret").AsString()
	assert.Equal(t, "my_secret", secretValue)
}

func TestAppRoot(t *testing.T) {
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
	buff := bytes.NewBuffer([]byte(yamlConfig1))
	provider := NewYAMLProviderFromReader(ioutil.NopCloser(buff))
	cs := &configStruct{}
	assert.NoError(t, provider.Get(Root).PopulateStruct(cs))
	assert.Equal(t, "yaml", provider.Scope(Root).Name())
	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "owner@service.com", cs.Owner)
}

func TestYAMLNode(t *testing.T) {
	buff := bytes.NewBuffer([]byte("a: b"))
	node := &yamlNode{value: make(map[interface{}]interface{})}
	err := unmarshalYAMLValue(ioutil.NopCloser(buff), &node.value)
	require.NoError(t, err)
	assert.Equal(t, "map[a:b]", node.String())
	assert.Equal(t, "map[interface {}]interface {}", node.Type().String())
}

func TestYamlNodeWithNil(t *testing.T) {
	provider := NewYAMLProviderFromFiles(false, nil)
	assert.NotNil(t, provider)
	assert.Panics(t, func() {
		_ = unmarshalYAMLValue(nil, nil)
	}, "Expected panic with nil inpout.")
}

func TestYamlNode_Callbacks(t *testing.T) {
	p := NewYAMLProviderFromFiles(false, nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}

func withYamlBytes(t *testing.T, yamlBytes []byte, f func(Provider)) {
	provider := NewProviderGroup("global", NewYAMLProviderFromBytes(yamlBytes))
	f(provider)
}

func TestMatchEmptyStruct(t *testing.T) {
	withYamlBytes(t, []byte(``), func(provider Provider) {
		es := emptystruct{}
		provider.Get("emptystruct").PopulateStruct(&es)
		empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
		assert.True(t, reflect.DeepEqual(empty, es))
	})
}

func TestMatchPopulatedEmptyStruct(t *testing.T) {
	withYamlBytes(t, emptyyaml, func(provider Provider) {
		es := emptystruct{}
		provider.Get("emptystruct").PopulateStruct(&es)
		empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
		assert.True(t, reflect.DeepEqual(empty, es))
	})
}

func TestPopulateStructWithPointers(t *testing.T) {
	withYamlBytes(t, pointerYaml, func(provider Provider) {
		ps := pointerStruct{}
		provider.Get("pointerStruct").PopulateStruct(&ps)
		assert.True(t, *ps.MyTrueBool)
		assert.False(t, *ps.MyFalseBool)
		assert.Equal(t, "hello", *ps.MyString)
	})
}

func TestNonExistingPopulateStructWithPointers(t *testing.T) {
	withYamlBytes(t, []byte(``), func(provider Provider) {
		ps := pointerStruct{}
		provider.Get("pointerStruct").PopulateStruct(&ps)
		assert.Nil(t, ps.MyTrueBool)
		assert.Nil(t, ps.MyFalseBool)
		assert.Nil(t, ps.MyString)
	})
}

func TestMapParsing(t *testing.T) {
	withYamlBytes(t, complexMapYaml, func(provider Provider) {
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
	withYamlBytes(t, simpleMapYaml, func(provider Provider) {
		ms := mapStruct{}
		provider.Get("mapStruct").PopulateStruct(&ms)
		assert.Equal(t, 1, ms.MyMap["one"])
		assert.Equal(t, 2, ms.MyMap["two"])
		assert.Equal(t, 3, ms.MyMap["three"])
		assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
	})
}

func TestMapParsingMapWithNonStringKeys(t *testing.T) {
	withYamlBytes(t, intKeyMapYaml, func(provider Provider) {
		ik := intKeyMapStruct{}
		err := provider.Get("intKeyMapStruct").PopulateStruct(&ik)
		assert.NoError(t, err)
		assert.Equal(t, "onetwothree", ik.IntKeyMap[123])
	})
}

func TestDurationParsing(t *testing.T) {
	withYamlBytes(t, durationYaml, func(provider Provider) {
		ds := durationStruct{}
		err := provider.Get("durationStruct").PopulateStruct(&ds)
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, ds.Seconds)
		assert.Equal(t, 20*time.Minute, ds.Minutes)
		assert.Equal(t, 30*time.Hour, ds.Hours)
	})
}

func TestParsingUnparsableDuration(t *testing.T) {
	withYamlBytes(t, unparsableDurationYaml, func(provider Provider) {
		ds := durationStruct{}
		err := provider.Get("durationStruct").PopulateStruct(&ds)
		assert.Error(t, err)
	})
}

func TestTypeOfTypes(t *testing.T) {
	withYamlBytes(t, typeStructYaml, func(provider Provider) {
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
	withYamlBytes(t, typeStructYaml, func(provider Provider) {
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
	withYamlBytes(t, typeStructYaml, func(provider Provider) {
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
	withYamlBytes(t, happyTextUnmarshallerYaml, func(provider Provider) {
		ds := duckTales{}
		err := provider.Get("duckTales").PopulateStruct(&ds)
		assert.NoError(t, err)
		assert.Equal(t, scrooge, ds.Protagonist)
		assert.Equal(t, launchpadMcQuack, ds.Pilot)
	})
}

func TestGrumpyTextUnMarshallerParsing(t *testing.T) {
	withYamlBytes(t, grumpyTextUnmarshallerYaml, func(provider Provider) {
		ds := duckTales{}
		err := provider.Get("darkwingDuck").PopulateStruct(&ds)
		assert.EqualError(t, err, "Unknown character: DarkwingDuck")
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
	provider := NewYAMLProviderFromBytes(yamlConfig1)
	c := provider.Get("modules.rpc.bind")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", c), "")
	}
	assert.NotPanics(t, f)
}

func TestArrayTypeNoPanic(t *testing.T) {
	// This test will panic if we treat array the same as slice.
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := struct {
		ID [6]int `yaml:"id"`
	}{}

	assert.NoError(t, provider.Get(Root).PopulateStruct(&cs))
}
