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

package main

import (
	"context"
	"sync"

	"go.uber.org/fx/examples/keyvalue/kv"
	kvs "go.uber.org/fx/examples/keyvalue/kv/keyvalueserver"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc"
)

type YarpcHandler struct {
	sync.RWMutex

	items map[string]string
}

func NewYarpcThriftHandler(_ service.Host, dispatcher *yarpc.Dispatcher) error {
	handler := &YarpcHandler{items: map[string]string{}}
	dispatcher.Register(kvs.New(handler))
	return nil
}

func (h *YarpcHandler) GetValue(ctx context.Context, key *string) (string, error) {
	h.RLock()
	defer h.RUnlock()

	if value, ok := h.items[*key]; ok {
		return value, nil
	}
	return "", &kv.ResourceDoesNotExist{Key: *key}
}

func (h *YarpcHandler) SetValue(ctx context.Context, key *string, value *string) error {
	h.Lock()

	h.items[*key] = *value
	h.Unlock()
	return nil
}
