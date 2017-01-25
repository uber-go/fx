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

import "go.uber.org/fx/service"

var gobEncoding = &GobEncoding{}

// Backend represents a task backend
type Backend interface {
	service.Module
	Encoder() Encoding
	Publish(message []byte, userContext map[string]string) error
	Consume() error
}

// NopBackend is a noop implementation of the Backend interface
type NopBackend struct{}

// Publish implements the Backend interface
func (b NopBackend) Publish(message []byte, userContext map[string]string) error {
	return nil
}

// Consume  implements the Backend interface
func (b NopBackend) Consume() error {
	return nil
}

// Encoder implements the Backend interface
func (b NopBackend) Encoder() Encoding {
	return &NopEncoding{}
}

// Type implements the Backend interface
func (b NopBackend) Type() string {
	return ""
}

// Name implements the Backend interface
func (b NopBackend) Name() string {
	return ""
}

// Start implements the Backend interface
func (b NopBackend) Start(ready chan<- struct{}) <-chan error {
	return make(chan error)
}

// Stop implements the Backend interface
func (b NopBackend) Stop() error {
	return nil
}

// IsRunning implements the Backend interface
func (b NopBackend) IsRunning() bool {
	return true
}

// InMemBackend is a noop implementation of the Backend interface
type InMemBackend struct {
	bufQueue chan []byte
}

// Publish implements the Backend interface
func (b *InMemBackend) Publish(message []byte, userContext map[string]string) error {
	b.bufQueue <- message
	return nil
}

// Consume  implements the Backend interface
func (b *InMemBackend) Consume() error {
	select {
	case v, ok := <-b.bufQueue:
		if ok {
			return Run(v)
		}
	default:
		// No value ready in channel, moving on
	}
	return nil
}

// Encoder implements the Backend interface
func (b *InMemBackend) Encoder() Encoding {
	return gobEncoding
}

// Type implements the Backend interface
func (b *InMemBackend) Type() string {
	return ""
}

// Name implements the Backend interface
func (b *InMemBackend) Name() string {
	return ""
}

// Start implements the Backend interface
func (b *InMemBackend) Start(ready chan<- struct{}) <-chan error {
	return make(chan error)
}

// Stop implements the Backend interface
func (b *InMemBackend) Stop() error {
	return nil
}

// IsRunning implements the Backend interface
func (b *InMemBackend) IsRunning() bool {
	return true
}
