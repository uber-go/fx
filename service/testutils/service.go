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

package testutils

import "go.uber.org/fx/service"

// WithService is a test helper to instantiate a service
func WithService(module service.ModuleCreateFunc, instance service.Observer, fn func(service.Owner)) {
	WithServices([]service.ModuleCreateFunc{module}, instance, fn)
}

// WithServices is a test helper to instantiate a service with multiple modules
func WithServices(modules []service.ModuleCreateFunc, observer service.Observer, fn func(service.Owner)) {
	if observer == nil {
		observer = service.ObserverStub()
	}

	svc, err := service.New(service.WithObserver(observer))
	if err != nil {
		panic(err)
	}
	err = svc.AddModules(modules...)

	if err != nil {
		panic(err)
	}
	fn(svc)
}
