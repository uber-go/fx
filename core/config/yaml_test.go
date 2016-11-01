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
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var yamlConfig1 = []byte(`
appid: keyvalue
desc: A simple keyvalue service
appowner: uberfx@uber.com
modules:
  rpc:
    bind: :28941
`)

func TestYamlSimple(t *testing.T) {
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	c := provider.GetValue("modules.rpc.bind")
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

	assert.True(t, provider.GetValue("").PopulateStruct(cs))

	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "uberfx@uber.com", cs.Owner)
}

type rpcStruct struct {
	Bind string `yaml:"bind"`
}

func TestYamlStructChild(t *testing.T) {
	provider := NewYAMLProviderFromBytes(yamlConfig1)

	cs := &rpcStruct{}

	assert.True(t, provider.GetValue("modules.rpc").PopulateStruct(cs))

	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	provider := NewYAMLProviderFromFiles(false, NewRelativeResolver("./testdata"), "base.yaml", "dev.yaml")

	baseValue := provider.GetValue("value").AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.GetValue("value_override").AsString()
	assert.Equal(t, "dev_setting", devValue)
}

func TestNewYamlProviderFromReader(t *testing.T) {
	buff := bytes.NewBuffer([]byte(yamlConfig1))
	provider := NewYamlProviderFromReader(ioutil.NopCloser(buff))
	cs := &configStruct{}
	assert.True(t, provider.GetValue("").PopulateStruct(cs))
	assert.Equal(t, "yaml", provider.Scope("").Name())
	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "uberfx@uber.com", cs.Owner)
}

func TestYamlNode(t *testing.T) {
	buff := bytes.NewBuffer([]byte("a: b"))
	node, err := newyamlNode(ioutil.NopCloser(buff))
	require.NoError(t, err)
	assert.Equal(t, "map[a:b]", node.String())
	assert.Equal(t, "map[interface {}]interface {}", node.Type().String())
}

func TestYamlNodeWithNil(t *testing.T) {
	provider := NewYAMLProviderFromFiles(false, nil)
	assert.NotNil(t, provider)
	assert.Panics(t, func() {
		_, _ = newyamlNode(nil)
	}, "Expected panic with nil inpout.")
}

func TestYamlNode_Callbacks(t *testing.T) {
	p := NewYAMLProviderFromFiles(false, nil)
	assert.NoError(t, p.RegisterChangeCallback("test", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))
}

type emptystruct struct {
	Slice []string
}

var emptyyaml = []byte(`
emptystruct:
  nonexist: true
`)

func TestPopulateMismatchedStruct(t *testing.T) {
	provider := NewProviderGroup("global", NewYAMLProviderFromBytes(emptyyaml))
	es := emptystruct{}
	empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
	assert.True(t, reflect.DeepEqual(empty, es))

	provider = NewProviderGroup("global", NewYAMLProviderFromBytes([]byte(``)))
	es = emptystruct{}
	provider.GetValue("emptystruct").PopulateStruct(&es)
	assert.True(t, reflect.DeepEqual(empty, es))
}
