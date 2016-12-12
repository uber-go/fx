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
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
)

// UnaryHandler is a wrapper for YARPC thrift.UnaryHandler
type UnaryHandler interface {
	Handle(ctx fx.Context, reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error)
}

// UnaryHandlerFunc calls the YARPC Handle with fx.Context
type UnaryHandlerFunc func(fx.Context, yarpc.ReqMeta, wire.Value) (thrift.Response, error)

// OnewayHandler is a wrapper for YARPC thrift.OnewayHandler
type OnewayHandler interface {
	HandleOneway(ctx fx.Context, reqMeta yarpc.ReqMeta, body wire.Value) error
}

// OnewayHandlerFunc calls the YARPC Handle with fx.Context
type OnewayHandlerFunc func(fx.Context, yarpc.ReqMeta, wire.Value) error

// Handle calls the caller HandlerFunc.
func (f UnaryHandlerFunc) Handle(ctx fx.Context, reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {
	return f(ctx, reqMeta, body)
}

// HandleOneway calls the caller OnewayHandlerFunc.
func (f OnewayHandlerFunc) HandleOneway(ctx fx.Context, reqMeta yarpc.ReqMeta, body wire.Value) error {
	return f(ctx, reqMeta, body)
}

// WrapUnary wraps the unary handler and returns implementation of thrift.UnaryHandler for yarpc calls
func WrapUnary(host service.Host, unaryHandlerFunc UnaryHandlerFunc) thrift.UnaryHandler {
	return &unaryHandlerWrapper{
		Host:             host,
		UnaryHandlerFunc: unaryHandlerFunc,
	}
}

type unaryHandlerWrapper struct {
	service.Host
	UnaryHandlerFunc
}

// Handle calls Handler.Handle(ctx, req, resp) and use the injected fx.context
func (hw *unaryHandlerWrapper) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {
	fxctx := fx.NewContext(ctx, hw.Host)
	return hw.UnaryHandlerFunc.Handle(fxctx, reqMeta, body)
}

// WrapOneway wraps the oneway handler and returns implementation of thrift.OnewayHandler for yarpc calls
func WrapOneway(host service.Host, onewayHandlerFunc OnewayHandlerFunc) thrift.OnewayHandler {
	return &onewayHandlerWrapper{
		Host:              host,
		OnewayHandlerFunc: onewayHandlerFunc,
	}
}

type onewayHandlerWrapper struct {
	service.Host
	OnewayHandlerFunc
}

// HandleOneway calls OnewayHandlerFunc.HandleOneway(ctx, req, resp) and use the injected fx.context
func (hw *onewayHandlerWrapper) HandleOneway(ctx context.Context, reqMeta yarpc.ReqMeta, body wire.Value) error {
	fxctx := fx.NewContext(ctx, hw.Host)
	return hw.OnewayHandlerFunc.HandleOneway(fxctx, reqMeta, body)
}
