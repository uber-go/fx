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

// A ModuleType is a human-friendly module type name
type ModuleType string

// A Module is the basic building block of an UberFx service
type Module interface {
	Initialize(ctx Context) error
	Type() string
	Name() string
	Start(ready chan<- struct{}) <-chan error
	Stop() error
	IsRunning() bool
}

// ModuleCreateInfo is used to configure module instantiation
// TODO(glib): this doesn't seem very heavily used. Should we keep it around?
// All modules have their own constructors, perhaps this is just the artifact
// of when all modules were initialized with []ModuleCreateInfo?
type ModuleCreateInfo struct {
	Name  string
	Roles []string
	Items map[string]interface{}
	Ctx   Context
}

// A ModuleCreateFunc handles instantiating modules from creation configuration
type ModuleCreateFunc func(ModuleCreateInfo) ([]Module, error)
