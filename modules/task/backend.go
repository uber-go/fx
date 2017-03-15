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

package task

import (
	"context"

	"go.uber.org/fx/service"
)

var gobEncoding = &GobEncoding{}

const (
	_initialized = iota
	_running
	_stopped
)

// Backend represents a task backend
type Backend interface {
	service.Module
	Encoder() Encoding
	Enqueue(ctx context.Context, message []byte) error
	ExecuteAsync() chan error
}

// NopBackend is a noop implementation of the Backend interface
type NopBackend struct{}

// Enqueue implements the Backend interface
func (b NopBackend) Enqueue(ctx context.Context, message []byte) error {
	return nil
}

// ExecuteAsync implements the Backend interface
func (b NopBackend) ExecuteAsync() chan error {
	return nil
}

// Encoder implements the Backend interface
func (b NopBackend) Encoder() Encoding {
	return &NopEncoding{}
}

// Start implements the Module interface
func (b NopBackend) Start() error {
	return nil
}

// Stop implements the Module interface
func (b NopBackend) Stop() error {
	return nil
}

// inMemBackend is an in-memory implementation of the Backend interface
type inMemBackend struct {
	service.Host
	bufQueue chan []byte
	errorCh  chan error
}

// NewManagedInMemBackend creates a new in memory backend, designed for use in tests
func NewManagedInMemBackend(host service.Host, cfg Config) Backend {
	b := NewInMemBackend(host)
	return &managedBackend{b, cfg}
}

// NewInMemBackend creates a new in memory backend, designed for use in tests
func NewInMemBackend(host service.Host) Backend {
	return &inMemBackend{
		bufQueue: make(chan []byte, 2),
		errorCh:  make(chan error, 1),
	}
}

// Encoder implements the Backend interface
func (b *inMemBackend) Encoder() Encoding {
	return gobEncoding
}

// Start implements the Module interface
func (b *inMemBackend) Start() error {
	return nil
}

// Enqueue implements the Backend interface
func (b *inMemBackend) Enqueue(ctx context.Context, message []byte) error {
	go func() {
		b.bufQueue <- message
	}()
	return nil
}

// ExecuteAsync implements the Backend interface
func (b *inMemBackend) ExecuteAsync() chan error {
	go func() {
		for msg := range b.bufQueue {
			// TODO(pedge): this was effectively not being handled and was a bug
			// The error channel passed in is the error channel used for start, which was
			// only read from once in host.startModules(), and this error was put into
			// the queue as a second error
			b.errorCh <- Run(context.Background(), msg)
		}
	}()
	return b.errorCh
}

// Stop implements the Module interface
func (b *inMemBackend) Stop() error {
	close(b.bufQueue)
	if b.errorCh != nil {
		close(b.errorCh)
	}
	return nil
}
