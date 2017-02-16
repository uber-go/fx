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

package rpc

import (
	"context"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/modules/rpc/internal/stats"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	"github.com/pkg/errors"
	"go.uber.org/yarpc/api/transport"
)

const _panicResponse = "Server Error"

type contextInboundMiddleware struct {
	service.Host
}

func (f contextInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	stopwatch := stats.RPCHandleTimer.
		Tagged(map[string]string{stats.TagProcedure: req.Procedure}).
		Timer(req.Procedure).
		Start()
	defer stopwatch.Stop()

	ctx = ulog.NewLogContext(ctx, ulog.Logger(ctx))
	return handler.Handle(ctx, req, resw)
}

type contextOnewayInboundMiddleware struct {
	service.Host
}

func (f contextOnewayInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	ctx = ulog.NewLogContext(ctx, ulog.Logger(ctx))
	return handler.HandleOneway(ctx, req)
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

func authorize(ctx context.Context, host service.Host) (context.Context, error) {
	if err := host.AuthClient().Authorize(ctx); err != nil {
		stats.RPCAuthFailCounter.Inc(1)
		ulog.Logger(ctx).Error(auth.ErrAuthorization, "error", err)
		// TODO(anup): GFM-255 update returned error to transport.BadRequestError (user error than server error)
		// https://github.com/yarpc/yarpc-go/issues/687
		return nil, err
	}
	return ctx, nil
}

type panicInboundMiddleware struct{}

func (p panicInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	defer panicRecovery(ctx)
	return handler.Handle(ctx, req, resw)
}

type panicOnewayInboundMiddleware struct{}

func (p panicOnewayInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	defer panicRecovery(ctx)
	return handler.HandleOneway(ctx, req)
}

func panicRecovery(ctx context.Context) {
	if err := recover(); err != nil {
		stats.RPCPanicCounter.Inc(1)
		ulog.Logger(ctx).Error("Panic recovered serving request", "error", errors.Errorf("panic in handler: %+v", err))
		// rethrow panic back to yarpc
		// before https://github.com/yarpc/yarpc-go/issues/734 fixed, throw a generic error.
		panic(_panicResponse)
	}
}
