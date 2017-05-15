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
	"os"
	"sync"
	"testing"
)

func TestNewExpandProvider(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { NewExpandProvider(nil, nil) })
	require.Panics(t, func() { NewExpandProvider(NewStaticProvider(""), nil) })
	p := NewExpandProvider(NewStaticProvider(""), os.Getenv)
	require.NotNil(t, p)
	assert.Equal(t, "expand", p.Name())
}

func TestExpandProvider_Get(t *testing.T) {
	t.Parallel()

	s := NewStaticProvider(map[string]interface{}{"a": "${1}", "b": 2})
	f := func(key string) string {
		require.Equal(t, "1", key)
		return "one"
	}

	p := NewExpandProvider(s, f)
	require.Equal(t, "one", p.Get("a").AsString())
	require.Equal(t, "2", p.Get("b").AsString())
	require.False(t, p.Get("c").HasValue())
}

func TestExpandProvider_RegisterChangeCallback(t *testing.T) {
	t.Parallel()

	d := NewMockDynamicProvider(map[string]interface{}{})
	f := func(key string) string {
		require.Equal(t, "1", key)
		return "one"
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	p := NewExpandProvider(d, f)
	err := p.RegisterChangeCallback("a", func(key string, provider string, data interface{}) {
		require.Equal(t, "a", key)
		require.Equal(t, "one", data)
		wg.Done()
	})

	require.NoError(t, err)

	err = p.RegisterChangeCallback("b", func(key string, provider string, data interface{}) {
		require.Equal(t, "b", key)
		require.Equal(t, "2", data)
		wg.Done()
	})

	require.NoError(t, err)

	d.Set("a", "${1}")
	d.Set("b", 2)

	wg.Wait()

	assert.NoError(t, p.UnregisterChangeCallback("a"))
}
