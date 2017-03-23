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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheProviderName(t *testing.T) {
	t.Parallel()

	c := NewCachedProvider(&MockDynamicProvider{})
	assert.Equal(t, `cached "MockDynamicProvider"`, c.Name())
}

func TestCachedProvider_ConstructorPanicsOnNil(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { NewCachedProvider(nil) })
}

func TestCachedProvider_GetNewValues(t *testing.T) {
	t.Parallel()

	m := &MockDynamicProvider{}
	p := NewCachedProvider(m)

	v := p.Get("cartoon")
	assert.False(t, v.HasValue())

	m.Set("cartoon", "Simpsons")
	v = v.Get(Root)
	require.True(t, v.HasValue())
	assert.Equal(t, "Simpsons", v.Value())

	ts := v.LastUpdated()
	m.Set("cartoon", "Futurama")
	assert.True(t, ts.Before(v.Get(Root).LastUpdated()))

	assert.Equal(t, p, v.provider)
}

func TestCachedProvider_ErrorToSetCallback(t *testing.T) {
	t.Parallel()

	m := &MockDynamicProvider{}
	p := NewCachedProvider(m)

	m.RegisterChangeCallback("cartoon", func(key, provider string, data interface{}) {})

	v := p.Get("cartoon")
	assert.False(t, v.HasValue())

	m.Set("cartoon", "Simpsons")
	v = v.Get(Root)
	require.False(t, v.HasValue())
}

func TestCachedProviderConcurrentUse(t *testing.T) {
	t.Parallel()

	wg := sync.WaitGroup{}

	m := &MockDynamicProvider{}
	p := NewCachedProvider(m)

	v := p.Get("cartoon")
	assert.False(t, v.HasValue())

	m.Set("cartoon", "Simpsons")
	wg.Add(4)
	get := func() {
		x := v.Get(Root)
		require.True(t, x.HasValue())
		wg.Done()
	}

	set := func() {
		m.Set("cartoon", "Jetsons")
		wg.Done()
	}

	go set()
	go get()
	go set()
	go get()

	wg.Wait()
}
