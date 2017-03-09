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

package service

// StubModuleProvider implements the ModuleProvider interface for testing.
type StubModuleProvider struct {
	NameVal   string
	CreateVal func(Host) (Module, error)
}

var _ ModuleProvider = &StubModuleProvider{}

// NewDefaultStubModuleProvider returns a new StubModuleProvider with an empty StubModule.
func NewDefaultStubModuleProvider() *StubModuleProvider {
	return NewStubModuleProvider("stubModule", &StubModule{})
}

// NewStubModuleProvider returns a new StubModuleProvider with the given name
// and create function from NewStubModuleCreateFunc.
func NewStubModuleProvider(name string, stubModule *StubModule) *StubModuleProvider {
	return &StubModuleProvider{NameVal: name, CreateVal: NewStubModuleCreateFunc(stubModule)}
}

// Name returns the module name
func (p *StubModuleProvider) Name() string {
	return p.NameVal
}

// Create creates a new Module.
func (p *StubModuleProvider) Create(host Host) (Module, error) {
	return p.CreateVal(host)
}

// A StubModule implements the Module interface for testing
type StubModule struct {
	Host       Host
	StartError error
	StopError  error
}

var _ Module = &StubModule{}

// DefaultStubModuleCreateFunc is a ModuleCreateFunc that returns a StubModule with only Host set.
var DefaultStubModuleCreateFunc = NewStubModuleCreateFunc(&StubModule{})

// NewStubModuleCreateFunc returns a new Module create function for a new StubModule.
//
// If stubModule is nil, this will return nil for the new Module.
//
// Host will be overwritten.
func NewStubModuleCreateFunc(stubModule *StubModule) func(Host) (Module, error) {
	return func(host Host) (Module, error) {
		if stubModule == nil {
			return nil, nil
		}
		stubModule.Host = host
		return stubModule, nil
	}
}

// NewStubModule generates a Module for use in testing
func NewStubModule(host Host) *StubModule {
	return &StubModule{Host: host}
}

// Start mimics startup
func (s *StubModule) Start() error { return s.StartError }

// Stop stops the module
func (s *StubModule) Stop() error { return s.StopError }
