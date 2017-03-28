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

package decorator

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// UnaryHandlerFunc represents the method call between two service layers.
type UnaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

// Handle adapter for UnaryHandlerFunc
func (u UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return u(ctx, req, resw)
}

// UnaryDecorator is a chainable behavior modifier for layer handlers.
type UnaryDecorator func(UnaryHandlerFunc) UnaryHandlerFunc

// BuildUnary wraps the provided layer with decorators
func BuildUnary(h UnaryHandlerFunc, m ...UnaryDecorator) UnaryHandlerFunc {
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

// UnaryHandlerWrap ...
func UnaryHandlerWrap(handler transport.UnaryHandler) UnaryHandlerFunc {
	return func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		return handler.Handle(ctx, req, resw)
	}
}

// OnewayHandlerFunc represents the method call between two service layers.
type OnewayHandlerFunc func(context.Context, *transport.Request) error

// HandleOneway adapter for OnewayHandlerFunc
func (u OnewayHandlerFunc) HandleOneway(ctx context.Context, req *transport.Request) error {
	return u(ctx, req)
}

// OnewayDecorator is a chainable behavior modifier for layer handlers.
type OnewayDecorator func(OnewayHandlerFunc) OnewayHandlerFunc

// BuildOneway wraps the provided layer with decorators
func BuildOneway(h OnewayHandlerFunc, m ...OnewayDecorator) OnewayHandlerFunc {
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

// OnewayHandlerWrap ...
func OnewayHandlerWrap(handler transport.OnewayHandler) OnewayHandlerFunc {
	return func(ctx context.Context, req *transport.Request) error {
		return handler.HandleOneway(ctx, req)
	}
}
