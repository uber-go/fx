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

// Package service is the Service Lifecycle.
//
// Service, being a bit of an overloaded term, requires some
// specific care to explain the various components in the
// servicepackage in UberFx.
//
//
// Instantiation
//
// Generally, you create a service in one of two ways:
//
// • The builder pattern, that is service.WithModules(...).Build()
//
// • Calling service.New() directly.
//
// The former is generally easier. We use the builder pattern in all examples, but
// New() is exported in case you'd like extra control over how your service is
// instantiated.
//
//
// If you **choose to** call service.New(), you need to call
// AddModules(...) to configure which modules you'd like to serve.
//
// Options
//
// Both the builder pattern and the New() function take a variadic Optionspattern, allowing you to pick and choose which components you'd like to
// override. As a common theme of UberFx, specifying zero options should give
// you a fully working application.
//
//
// Once you have a service, you call .Start() on it to begin receiving requests.
//
// Start(bool) comes in two variants: a blocking version and a non-blocking
// version. In our sample apps, we use the blocking version (
// svc.Start(true)) and
// yield control to the service lifecycle manager. If you wish to do other things
// after starting your service, you may pass
// false and use the return values of
// svc.Start(bool) to listen on channels and trigger manual shutdowns.
//
//
package service
