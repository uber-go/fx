package metrics

import (
	"testing"

	"github.com/uber-go/uberfx/core/config"
	"github.com/stretchr/testify/assert"
)

const yamlMetricsTags = `
metrics:
  tags:
    foo: bar
    baz: boing
`

var globalConfig config.ConfigurationProvider

func init() {
	globalConfig = config.Global()
}

func reset() {
	config.SetGlobal(globalConfig, true)
}

func TestTags(t *testing.T) {
	defer reset()

	config.SetGlobal(config.NewYamlProviderFromString(yamlMetricsTags), true)

	scope := Global(false)

	// how to know?
	assert.NotNil(t, scope)
}
