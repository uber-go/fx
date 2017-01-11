// Copyright (c) 2016 Uber Technologies, Inc.
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

package service

// A StubModule implements the Module interface for testing
type StubModule struct {
	Host             Host
	InitError        error
	TypeVal, NameVal string
	StartError       error
	StopError        error
	Running          bool
}

var _ Module = &StubModule{}

// NewStubModule generates a Module for use in testing
func NewStubModule(host Host) *StubModule {
	return &StubModule{Host: host}
}

// Start mimics startup
func (s *StubModule) Start(ready chan<- struct{}) <-chan error {
	errs := make(chan error, 1)
	errs <- s.StartError
	ready <- struct{}{}
	s.Running = true
	return errs
}

// Type returns the type of the module
func (s *StubModule) Type() string { return s.TypeVal }

// Name returns the name of the module
func (s *StubModule) Name() string { return s.NameVal }

// IsRunning returns the current running state
func (s *StubModule) IsRunning() bool { return s.Running }

// Stop stops the module
func (s *StubModule) Stop() error { return s.StopError }
