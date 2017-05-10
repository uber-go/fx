package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupProvider_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "lookup", NewLookupProvider(nil).Name())
}


func TestLookupProvider_Get(t *testing.T) {
	t.Parallel()

	f := func(key string) (interface{}, bool) {
		if key == "Life was like a box of chocolates" {
			return "you never know what you're gonna get", true
		}

		return nil, false
	}

	p := NewLookupProvider(f)
	assert.Equal(t, "you never know what you're gonna get",
		p.Get("Life was like a box of chocolates").AsString())

	assert.Equal(t, false, p.Get("Forrest Gump").HasValue())
}
