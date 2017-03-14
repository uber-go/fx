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

	"github.com/pkg/errors"
)

// MockDynamicProvider is simple implementation of Provider that can be used to test dynamic features.
// It is safe to use with multiple go routines, but doesn't support nested objects.
type MockDynamicProvider struct {
	sync.RWMutex
	data      map[string]interface{}
	callBacks map[string]ChangeCallback
}

// NewMockDynamicProvider returns a new MockDynamicProvider
func NewMockDynamicProvider(data map[string]interface{}) *MockDynamicProvider {
	return &MockDynamicProvider{
		data: data,
	}
}

// Name is MockDynamicProvider
func (s *MockDynamicProvider) Name() string {
	return "MockDynamicProvider"
}

// Get returns a value in the map.
func (s *MockDynamicProvider) Get(key string) Value {
	s.RLock()
	defer s.RUnlock()

	val, found := s.data[key]
	return NewValue(s, key, val, found, GetType(val), nil)
}

// Set value to specific key and then calls a corresponding callback.
func (s *MockDynamicProvider) Set(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()

	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	s.data[key] = value
	if cb, ok := s.callBacks[key]; ok {
		cb(key, s.Name(), value)
	}
}

// RegisterChangeCallback registers a callback to be called when a value associated with a key will change.
func (s *MockDynamicProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	s.Lock()
	defer s.Unlock()

	if s.callBacks == nil {
		s.callBacks = make(map[string]ChangeCallback)
	}

	if _, ok := s.callBacks[key]; ok {
		return errors.New("callback already registered for the key: " + key)
	}

	s.callBacks[key] = callback
	return nil
}

// UnregisterChangeCallback removes a callback associated with a token.
func (s *MockDynamicProvider) UnregisterChangeCallback(token string) error {
	s.Lock()
	defer s.Unlock()

	if s.callBacks == nil {
		s.callBacks = make(map[string]ChangeCallback)
	}

	if _, ok := s.callBacks[token]; !ok {
		return errors.New("there is no registered callback for token: " + token)
	}

	delete(s.callBacks, token)
	return nil
}
