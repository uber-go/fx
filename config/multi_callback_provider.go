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

type multiCallbackProvider struct {
	sync.RWMutex
	cb map[string][]ChangeCallback
	Provider
}

// NewMultiCallbackProvider returns a provider that lets you to have multiple callbacks for a given Provider.
// All registered callbacks are going to be executed in the order they were registered.
// UnregisterCallback will unregister the most recently registered callback.
func NewMultiCallbackProvider(p Provider) Provider {
	if p == nil {
		return nil
	}

	return &multiCallbackProvider{
		cb:       make(map[string][]ChangeCallback),
		Provider: p,
	}
}

func (s *multiCallbackProvider) Name() string {
	return fmt.Sprintf("multiCallbackProvider %q", s.Provider.Name())
}

func (s *multiCallbackProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	s.Lock()
	defer s.Unlock()
	if val, exist := s.cb[key]; exist {
		s.cb[key] = append(val, callback)
		return nil
	}

	err := s.Provider.RegisterChangeCallback(key, func(key string, provider string, data interface{}) {
		s.RLock()
		defer s.RUnlock()

		for _, cb := range s.cb[key] {
			cb(key, provider, data)
		}
	})

	if err != nil {
		return err
	}

	s.cb[key] = []ChangeCallback{callback}
	return nil
}

func (s *multiCallbackProvider) UnregisterChangeCallback(token string) error {
	s.Lock()
	defer s.Unlock()

	if stack, ok := s.cb[token]; ok {
		if len(stack) > 1 {
			stack = stack[:len(stack)-1]
			return nil
		}

		delete(s.cb, token)
	}

	return s.Provider.UnregisterChangeCallback(token)
}
