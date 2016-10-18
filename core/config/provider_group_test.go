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

func TestProviderGroup(t *testing.T) {
	pg := NewProviderGroup("test-group", NewYAMLProviderFromBytes([]byte(`id: test`)))
	assert.Equal(t, "test-group", pg.Name())
	assert.Equal(t, "test", pg.GetValue("id").AsString())
	// TODO this should not require a cast GFM-74
	assert.Empty(t, pg.(providerGroup).RegisterChangeCallback("", nil))
	assert.False(t, pg.(providerGroup).UnregisterChangeCallback(""))
}

func TestProviderGroupScope(t *testing.T) {
	data := map[string]interface{}{"hello.world": 42}
	pg := NewProviderGroup("test-group", StaticProvider(data))
	assert.Equal(t, 42, pg.Scope("hello").GetValue("world").AsInt())
}
