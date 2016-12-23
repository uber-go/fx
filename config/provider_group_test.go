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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderGroup(t *testing.T) {
	pg := NewProviderGroup("test-group", NewYAMLProviderFromBytes([]byte(`id: test`)))
	assert.Equal(t, "test-group", pg.Name())
	assert.Equal(t, "test", pg.Get("id").AsString())
	// TODO this should not require a cast GFM-74
	assert.Empty(t, pg.(providerGroup).RegisterChangeCallback(Root, nil))
	assert.Nil(t, pg.(providerGroup).UnregisterChangeCallback(Root))
}

func TestProviderGroupScope(t *testing.T) {
	data := map[string]interface{}{"hello.world": 42}
	pg := NewProviderGroup("test-group", NewStaticProvider(data))
	assert.Equal(t, 42, pg.Scope("hello").Get("world").AsInt())
}

func TestCallbacks_WithDynamicProvider(t *testing.T) {
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data))
	mock = mock.(providerGroup).WithProvider(newMockDynamicProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())
	assert.Equal(t, fmt.Errorf("registration error"), mock.RegisterChangeCallback("mockcall", nil))
	assert.Equal(t, fmt.Errorf("unregiser error"), mock.UnregisterChangeCallback("mock"))
}

func TestCallbacks_WithoutDynamicProvider(t *testing.T) {
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data))
	mock = mock.(providerGroup).WithProvider(NewStaticProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())
	assert.NoError(t, mock.RegisterChangeCallback("mockcall", nil))
	assert.NoError(t, mock.UnregisterChangeCallback("mock"))
}

type mockDynamicProvider struct {
	data map[string]interface{}
}

// StaticProvider should only be used in tests to isolate config from your environment
func newMockDynamicProvider(data map[string]interface{}) Provider {
	return &mockDynamicProvider{
		data: data,
	}
}

func (*mockDynamicProvider) Name() string {
	return "mock"
}

func (s *mockDynamicProvider) Get(key string) Value {
	val, found := s.data[key]
	return NewValue(s, key, val, found, GetType(val), nil)
}

func (s *mockDynamicProvider) Scope(prefix string) Provider {
	return NewScopedProvider(prefix, s)
}

func (s *mockDynamicProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
	return fmt.Errorf("registration error")
}

func (s *mockDynamicProvider) UnregisterChangeCallback(token string) error {
	return fmt.Errorf("unregiser error")
}
