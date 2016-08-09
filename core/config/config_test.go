package config

import (
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

const nest1 = `
id1: 1
id2: 2
`

type root struct {
	ID        int      `yaml:"id"`
	Names     []string `yaml:"names"`
	Nested    nested   `yaml:"n1"`
	NestedPtr *nested  `yaml:"nptr"`
}

const nestedYaml = `
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
`

const structArrayYaml = `
things:
  - id1: 0
  - id1: 1
  - id1: 2
`

const yamlConfig2 = `
appid: keyvalue
desc: A simple keyvalue service
appowner: uberfx@uber.com
modules:
  rpc:
    bind: :28941
`

type arrayOfStructs struct {
	Things []nested `yaml:"things"`
}

func TestDirectAccess(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYamlProviderFromString(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	v := provider.GetValue("n1.id1", "xxx")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())

	v2 := provider.GetValue("n1.id2", "xxx")

	assert.True(t, v2.HasValue())
	assert.Equal(t, "-1", v2.Value())
}

func TestOverrideSimple(t *testing.T) {

	provider := NewProviderGroup(
		"test",
		NewYamlProviderFromString(yamlConfig2),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	rpc := &rpcStruct{}
	v := provider.GetValue("modules.rpc", nil)
	assert.True(t, v.HasValue())
	v.PopulateStruct(rpc)
	assert.Equal(t, ":8888", rpc.Bind)

}

func TestNestedStructs(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYamlProviderFromString(nestedYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	str := &root{}

	v := provider.GetValue("", nil)

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
		NewYamlProviderFromString(structArrayYaml),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)

	target := &arrayOfStructs{}

	v := provider.GetValue("", nil)

	assert.True(t, v.HasValue())
	assert.True(t, v.PopulateStruct(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, -2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	provider := NewProviderGroup(
		"test",
		NewYamlProviderFromString(nest1),
		NewEnvProvider(defaultEnvPrefix, mapEnvironmentProvider{values: env}),
	)
	target := &nested{}
	v := provider.GetValue("", nil)
	assert.True(t, v.HasValue())
	assert.True(t, v.PopulateStruct(target))
	assert.Equal(t, "default_name", target.Name)
}
