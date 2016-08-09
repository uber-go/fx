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

	c := provider.MustGetValue("modules.rpc.bind")
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

	assert.True(t, provider.GetValue("", nil).PopulateStruct(cs))

	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "uberfx@uber.com", cs.Owner)
}

type rpcStruct struct {
	Bind string `yaml:"bind"`
}

func TestYamlStructChild(t *testing.T) {
	provider := NewYamlProviderFromString(yamlConfig1)

	cs := &rpcStruct{}

	assert.True(t, provider.GetValue("modules.rpc", nil).PopulateStruct(cs))

	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	provider := NewYamlProviderFromFiles(false, NewRelativeResolver("./test"), "base.yaml", "dev.yaml")

	baseValue := provider.GetValue("value", nil).AsString()
	assert.Equal(t, "base_only", baseValue)

	devValue := provider.GetValue("value_override", nil).AsString()
	assert.Equal(t, "dev_setting", devValue)
}
