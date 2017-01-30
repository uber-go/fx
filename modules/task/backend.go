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
	"fmt"
	"time"

	"go.uber.org/fx/service"
)

var gobEncoding = &GobEncoding{}

// Backend represents a task backend
type Backend interface {
	service.Module
	Encoder() Encoding
	Publish(message []byte, userContext map[string]string) error
}

// NopBackend is a noop implementation of the Backend interface
type NopBackend struct{}

// Publish implements the Backend interface
func (b NopBackend) Publish(message []byte, userContext map[string]string) error {
	return nil
}

// Encoder implements the Backend interface
func (b NopBackend) Encoder() Encoding {
	return &NopEncoding{}
}

// Name implements the Module interface
func (b NopBackend) Name() string {
	return "nop"
}

// Start implements the Module interface
func (b NopBackend) Start(ready chan<- struct{}) <-chan error {
	return make(chan error)
}

// Stop implements the Module interface
func (b NopBackend) Stop() error {
	return nil
}

// IsRunning implements the Module interface
func (b NopBackend) IsRunning() bool {
	return true
}

// inMemBackend is an in-memory implementation of the Backend interface
type inMemBackend struct {
	bufQueue  chan []byte
	isRunning bool
}

// NewInMemBackend creates a new in memory backend, designed for use in tests
func NewInMemBackend() Backend {
	return &inMemBackend{bufQueue: make(chan []byte, 2)}
}

// Encoder implements the Backend interface
func (b *inMemBackend) Encoder() Encoding {
	return gobEncoding
}

// Name implements the Module interface
func (b *inMemBackend) Name() string {
	return "inMem"
}

// Start implements the Module interface
func (b *inMemBackend) Start(ready chan<- struct{}) <-chan error {
	b.isRunning = true
	errorCh := make(chan error, 2)
	go func(buf chan []byte) {
		for {
			select {
			case msg := <-buf:
				errorCh <- Run(msg)
			case <-time.After(time.Second):
				fmt.Println("Timed out after 1 second")
			}
		}
	}(b.bufQueue)
	return errorCh
}

// Publish implements the Backend interface
func (b *inMemBackend) Publish(message []byte, userContext map[string]string) error {
	go func() {
		b.bufQueue <- message
	}()
	return nil
}

// Stop implements the Module interface
func (b *inMemBackend) Stop() error {
	b.isRunning = false
	close(b.bufQueue)
	return nil
}

// IsRunning implements the Module interface
func (b *inMemBackend) IsRunning() bool {
	return b.isRunning
}
