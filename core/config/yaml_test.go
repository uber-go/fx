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
	"testing"

	"github.com/stretchr/testify/assert"
)

const yamlConfig1 = `
appid: keyvalue
desc: A simple keyvalue service
appowner: uberfx@uber.com
modules:
  rpc:
    bind: :28941
`

func TestYamlSimple(t *testing.T) {
	provider := NewYamlProviderFromString(yamlConfig1)

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
	provider := NewYamlProviderFromString(yamlConfig1)

	cs := &configStruct{}

	assert.True(t, provider.GetValue("").PopulateStruct(cs))

	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "uberfx@uber.com", cs.Owner)
}

type rpcStruct struct {
	Bind string `yaml:"bind"`
}

func TestYamlStructChild(t *testing.T) {
	provider := NewYamlProviderFromString(yamlConfig1)

	cs := &rpcStruct{}

	assert.True(t, provider.GetValue("modules.rpc").PopulateStruct(cs))

	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	provider := NewYamlProviderFromFiles(false, NewRelativeResolver("./test"), "base.yaml", "dev.yaml")

	baseValue := provider.GetValue("value").AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.GetValue("value_override").AsString()
	assert.Equal(t, "dev_setting", devValue)
}
