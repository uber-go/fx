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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestNewStackCallbackProvider_ForNilProvider(t *testing.T) {
	t.Parallel()

	assert.Nil(t, NewStackCallbackProvider(nil))
}

func TestStackCallbackProvider_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `stackCallbackProvider "static"`, NewStackCallbackProvider(NewStaticProvider(42)).Name())
}

func TestStackCallbackProvider_RegisterChangeCallback(t *testing.T) {
	t.Parallel()

	m := NewMockDynamicProvider(nil)
	s := NewStackCallbackProvider(m)
	require.NotNil(t, s)

	wg := sync.WaitGroup{}
	totalCalls := 0
	wg.Add(2)
	register := func() {
		s.RegisterChangeCallback("key", func(key string, provider string, data interface{}) {
			totalCalls++
			assert.Equal(t, "secret", data)
		})

		wg.Done()
	}

	go register()
	go register()

	wg.Wait()

	m.Set("key", "secret")
	assert.Equal(t, 2, totalCalls)
}

func TestStackCallbackProvider_UnregisterChangeCallbackRace(t *testing.T) {
	t.Parallel()

	m := NewMockDynamicProvider(nil)
	s := NewStackCallbackProvider(m)
	require.NotNil(t, s)

	cb := func(key string, provider string, data interface{}) {
		assert.Equal(t, "bruce wayne", data)
	}

	require.NoError(t, s.RegisterChangeCallback("batman", cb))
	require.NoError(t, s.RegisterChangeCallback("batman", cb))

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		require.NoError(t, s.UnregisterChangeCallback("batman"))
		wg.Done()
	}()

	go func() {
		m.Set("batman", "bruce wayne")
		wg.Done()
	}()

	wg.Wait()
}

func TestStackCallbackProvider_ErrorToRegister(t *testing.T) {
	t.Parallel()

	m := NewMockDynamicProvider(nil)
	s := NewStackCallbackProvider(m)
	require.NotNil(t, s)

	require.NoError(t, m.RegisterChangeCallback("robin", nil))
	err := s.RegisterChangeCallback("robin", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "callback already registered for the key: robin")
}

func TestStackCallbackProvider_ErrorToUnregister(t *testing.T) {
	t.Parallel()

	s := NewStackCallbackProvider(NewMockDynamicProvider(nil))
	require.NotNil(t, s)

	require.NoError(t, s.RegisterChangeCallback("robin", nil))
	require.NoError(t, s.UnregisterChangeCallback("robin"))

	err := s.UnregisterChangeCallback("robin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "there is no registered callback for token: robin")
}
