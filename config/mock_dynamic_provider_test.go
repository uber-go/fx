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

func TestMockDynamicProvider_GetAndSet(t *testing.T) {
	t.Parallel()

	p := NewMockDynamicProvider(map[string]interface{}{"goofy": "empty"})
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		v := p.Get("goofy")
		assert.True(t, v.HasValue())
		wg.Done()
	}()

	go func() {
		p.Set("goofy", "gopher")
		v := p.Get("goofy")
		assert.Equal(t, "gopher", v.Value())
		wg.Done()
	}()

	wg.Wait()
}

func TestMockDynamicProvider_RegisterChangeCallback(t *testing.T) {
	t.Parallel()

	p := NewMockDynamicProvider(map[string]interface{}{"goofy": "empty"})

	wg := sync.WaitGroup{}
	wg.Add(4)
	p.RegisterChangeCallback("goofy", func(key string, provider string, data interface{}) {
		require.Equal(t, "goofy", key)
		require.Equal(t, p.Name(), provider)
		assert.NotEqual(t, "empty", data)
		wg.Done()
	})

	go func() {
		p.Set("goofy", "gopher")
		wg.Done()
	}()

	go func() {
		p.Set("goofy", "gofer")
		wg.Done()
	}()

	wg.Wait()
}

func TestMockDynamicProvider_UnregisterChangeCallback(t *testing.T) {
	t.Parallel()

	p := NewMockDynamicProvider(map[string]interface{}{"goofy": ""})

	require.EqualError(t,
		p.UnregisterChangeCallback("goofy"),
		"there is no registered callback for token: goofy")

	p.RegisterChangeCallback("goofy", func(key string, provider string, data interface{}) {
		require.Fail(t, "should not be called")
	})

	errChan := make(chan error, 1)
	unregister := func() {
		if err := p.UnregisterChangeCallback("goofy"); err != nil {
			errChan <- err
		}
	}

	go unregister()
	go unregister()

	assert.EqualError(t, <-errChan, "there is no registered callback for token: goofy")
}
