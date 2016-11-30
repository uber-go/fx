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

package rpc

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/transport"
)

// Handler is a context-aware extension of transport.Handler.
type Handler interface {
	// Handle the given request, writing the response to the given ResponseWriter.
	Handle(ctx fx.Context, req *transport.Request, resw transport.ResponseWriter) error
}

type handlerWrapper struct {
	service.Host
	handler Handler
}

// Handle calls Handler.Handle(ctx, req, resp) and use injected fx.context
func (h *handlerWrapper) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	fxctx := fx.NewContext(ctx, h.Host)
	return h.handler.Handle(fxctx, req, resw)
}

// WrapHandler wraps the handler and returns implementation of transport.Handler for yarpc calls
func WrapHandler(host service.Host, handler Handler) transport.Handler {
	return &handlerWrapper{
		Host:    host,
		handler: handler,
	}
}
