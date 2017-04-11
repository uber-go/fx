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
	"fmt"
	"sync"
)

type cachedProvider struct {
	sync.RWMutex
	cache map[string]Value

	Provider
}

// NewCachedProvider returns a provider, that caches values of the underlying Provider p.
// It also subscribes for changes for all keys that ever retrieved from the provider.
// If the underlying provider fails to register callback for a particular value, it will
// return the underlying error wrapped in Value.
func NewCachedProvider(p Provider) Provider {
	if p == nil {
		panic("Received a nil provider")
	}

	return &cachedProvider{
		Provider: p,
		cache:    make(map[string]Value),
	}
}

func (p *cachedProvider) Name() string {
	return fmt.Sprintf("cached %q", p.Provider.Name())
}

// Retrieves a Value, caches it internally and subscribe to changes via RegisterCallback.
// The value is cached only if it is found.
func (p *cachedProvider) Get(key string) Value {
	p.RLock()
	if v, ok := p.cache[key]; ok {
		p.RUnlock()
		return v
	}

	p.RUnlock()
	err := p.Provider.RegisterChangeCallback(key, func(key string, provider string, data interface{}) {
		p.Lock()
		p.cache[key] = NewValue(p, key, data, true, GetType(data), nil)
		p.Unlock()
	})

	if err != nil {
		return NewValue(p, key, err, false, GetType(err), nil)
	}

	v := p.Provider.Get(key)
	v.provider = p
	p.Lock()
	p.cache[key] = v
	p.Unlock()

	return v
}

// No need to register a callback, all the values are fresh.
func (p *cachedProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	return nil
}

// No need to unregister a callback, because nothing was registered.
func (p *cachedProvider) UnregisterChangeCallback(token string) error {
	return nil
}
