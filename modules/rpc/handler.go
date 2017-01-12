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
	"go.uber.org/fx/auth"
	"go.uber.org/fx/internal/fxcontext"
	"go.uber.org/fx/service"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
)

// UnaryHandler is a wrapper for YARPC thrift.UnaryHandler
type UnaryHandler interface {
	Handle(ctx fx.Context, body wire.Value) (thrift.Response, error)
}

// UnaryHandlerFunc calls the YARPC Handle with fx.Context
type UnaryHandlerFunc func(fx.Context, wire.Value) (thrift.Response, error)

// Handle calls the caller HandlerFunc.
func (f UnaryHandlerFunc) Handle(ctx fx.Context, body wire.Value) (thrift.Response, error) {
	return f(ctx, body)
}

// WrapUnary wraps the unary handler and returns a thrift.UnaryHandlerFunc for yarpc calls
// TODO(anup): GFM-255 Remove host parameter once updated yarpc plugin is imported in the repo
func WrapUnary(h UnaryHandlerFunc) thrift.UnaryHandler {
	return func(ctx context.Context, body wire.Value) (thrift.Response, error) {
		fxctx := &fxcontext.Context{
			Context: ctx,
		}
		return h.Handle(fxctx, body)
	}
}

// OnewayHandler is a wrapper for YARPC thrift.OnewayHandler
type OnewayHandler interface {
	HandleOneway(ctx fx.Context, body wire.Value) error
}

// OnewayHandlerFunc calls the YARPC Handle with fx.Context
type OnewayHandlerFunc func(fx.Context, wire.Value) error

// HandleOneway calls the caller OnewayHandlerFunc.
func (f OnewayHandlerFunc) HandleOneway(ctx fx.Context, body wire.Value) error {
	return f(ctx, body)
}

// WrapOneway wraps the oneway handler and returns a thrift.OnewayHandlerFunc for yarpc calls
// TODO(anup): GFM-255 Remove host parameter once updated yarpc plugin is imported in the repo
func WrapOneway(h OnewayHandlerFunc) thrift.OnewayHandler {
	return func(ctx context.Context, body wire.Value) error {
		fxctx := &fxcontext.Context{
			Context: ctx,
		}
		return h.HandleOneway(fxctx, body)
	}
}

type fxContextInboundMiddleware struct {
	service.Host
}

func (f fxContextInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	fxctx := fxcontext.New(ctx, f.Host)
	return handler.Handle(fxctx, req, resw)
}

type fxContextOnewayInboundMiddleware struct {
	service.Host
}

func (f fxContextOnewayInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	fxctx := fxcontext.New(ctx, f.Host)
	return handler.HandleOneway(fxctx, req)
}

type authInboundMiddleware struct {
	service.Host
}

func (a authInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	fxctx, err := authorize(ctx, a.Host)
	if err != nil {
		return err
	}
	return handler.Handle(fxctx, req, resw)
}

type authOnewayInboundMiddleware struct {
	service.Host
}

func (a authOnewayInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	fxctx, err := authorize(ctx, a.Host)
	if err != nil {
		return err
	}
	return handler.HandleOneway(fxctx, req)
}

func authorize(ctx context.Context, host service.Host) (fx.Context, error) {
	fxctx := &fxcontext.Context{
		Context: ctx,
	}
	if err := host.AuthClient().Authorize(fxctx); err != nil {
		host.Metrics().SubScope("rpc").SubScope("auth").Counter("fail").Inc(1)
		fxctx.Logger().Error(auth.ErrAuthorization, "error", err)
		// TODO(anup): GFM-255 update returned error to transport.BadRequestError (user error than server error)
		return nil, err
	}
	return fxctx, nil
}
