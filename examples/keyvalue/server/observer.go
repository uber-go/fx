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

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

// Observer receives callbacks during various service lifecycle events
type Observer struct {
	service.HostContainer

	ServiceConfig serviceConfig
	someFlag      bool
}

// OnInit is called during service init process. Returning an error halts the init?
func (o *Observer) OnInit(svc service.Host) error {
	ulog.Logger(context.Background()).Info(
		"Received service init callback",
		"service_name", o.Name(),
		"some_number", o.ServiceConfig.SomeNumber,
	)

	return nil
}

// OnStateChange is called when service changes state
func (o *Observer) OnStateChange(old service.State, new service.State) {}

// OnShutdown is called during shutdown
func (o *Observer) OnShutdown(reason service.Exit) {}

// OnCriticalError is called during critical errors
func (o *Observer) OnCriticalError(err error) bool { return false }

// Validate that Observer satisfies the interface
var _ service.Observer = &Observer{}
