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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderGroup(t *testing.T) {
	t.Parallel()
	pg := NewProviderGroup("test-group", NewYAMLProviderFromBytes([]byte(`id: test`)))
	assert.Equal(t, "test-group", pg.Name())
	assert.Equal(t, "test", pg.Get("id").AsString())
	// TODO this should not require a cast GFM-74
	assert.Empty(t, pg.(providerGroup).RegisterChangeCallback(Root, nil))
	assert.Nil(t, pg.(providerGroup).UnregisterChangeCallback(Root))
}

func TestProviderGroupScope(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello.world": 42}
	pg := NewProviderGroup("test-group", NewStaticProvider(data))
	assert.Equal(t, 42, pg.Scope("hello").Get("world").AsInt())
}

func TestCallbacks_WithDynamicProvider(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data))
	mock = mock.(providerGroup).WithProvider(newMockDynamicProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())

	require.NoError(t, mock.RegisterChangeCallback("mockcall", nil))
	assert.EqualError(t,
		mock.RegisterChangeCallback("mockcall", nil),
		"Callback already registered for the key: mockcall")

	assert.EqualError(t,
		mock.UnregisterChangeCallback("mock"),
		"There is no registered callback for token: mock")
}

func TestCallbacks_WithoutDynamicProvider(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data))
	mock = mock.(providerGroup).WithProvider(NewStaticProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())
	assert.NoError(t, mock.RegisterChangeCallback("mockcall", nil))
	assert.NoError(t, mock.UnregisterChangeCallback("mock"))
}

func TestCallbacks_WithScopedProvider(t *testing.T) {
	t.Parallel()
	mock := &mockDynamicProvider{}
	mock.Set("uber.fx", "go-lang")
	scope := NewScopedProvider("uber", mock)

	callCount := 0
	cb := func(key string, provider string, configdata interface{}) {
		callCount++
	}

	require.NoError(t, scope.RegisterChangeCallback("fx", cb))
	mock.Set("uber.fx", "register works!")

	val := scope.Get("fx").AsString()
	require.Equal(t, "register works!", val)
	assert.Equal(t, 1, callCount)

	require.NoError(t, scope.UnregisterChangeCallback("fx"))
	mock.Set("uber.fx", "unregister works too!")

	val = scope.Get("fx").AsString()
	require.Equal(t, "unregister works too!", val)
	assert.Equal(t, 1, callCount)
}

func TestScope_WithScopedProvider(t *testing.T) {
	t.Parallel()
	mock := &mockDynamicProvider{}
	mock.Set("uber.fx", "go-lang")
	scope := NewScopedProvider("", mock)
	require.Equal(t, "go-lang", scope.Get("uber.fx").AsString())
	require.False(t, scope.Get("uber").HasValue())

	base := scope.Scope("uber")
	require.Equal(t, "go-lang", base.Get("fx").AsString())
	require.False(t, base.Get("").HasValue())

	uber := base.Scope("")
	require.Equal(t, "go-lang", uber.Get("fx").AsString())
	require.False(t, uber.Get("").HasValue())

	fx := uber.Scope("fx")
	require.Equal(t, "go-lang", fx.Get("").AsString())
	require.False(t, fx.Get("fx").HasValue())
}

type mockDynamicProvider struct {
	data      map[string]interface{}
	callBacks map[string]ConfigurationChangeCallback
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

func (s *mockDynamicProvider) Set(key string, value interface{}) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	s.data[key] = value
	if cb, ok := s.callBacks[key]; ok {
		cb(key, s.Name(), "randomConfig")
	}
}

func (s *mockDynamicProvider) Scope(prefix string) Provider {
	return NewScopedProvider(prefix, s)
}

func (s *mockDynamicProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
	if s.callBacks == nil {
		s.callBacks = make(map[string]ConfigurationChangeCallback)
	}

	if _, ok := s.callBacks[key]; ok {
		return errors.New("Callback already registered for the key: " + key)
	}

	s.callBacks[key] = callback
	return nil
}

func (s *mockDynamicProvider) UnregisterChangeCallback(token string) error {
	if s.callBacks == nil {
		s.callBacks = make(map[string]ConfigurationChangeCallback)
	}

	if _, ok := s.callBacks[token]; !ok {
		return errors.New("There is no registered callback for token: " + token)
	}

	delete(s.callBacks, token)
	return nil
}
