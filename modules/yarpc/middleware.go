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

package yarpc

import (
	"context"
	"sync"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"

	"github.com/pkg/errors"
	"go.uber.org/yarpc/api/transport"
)

const _panicResponse = "Server Error"

var (
	mu sync.Mutex
)

// TransportUnaryMiddleware keeps all the decorator layers defined in the configuration
type TransportUnaryMiddleware struct {
	procedures map[string][]decorator.UnaryDecorator
	decorators map[string]transport.UnaryHandler
}

// Handle all decorators for Unary transport
func (l TransportUnaryMiddleware) Handle(
	ctx context.Context,
	req *transport.Request,
	resw transport.ResponseWriter,
	handler transport.UnaryHandler,
) error {
	if _, ok := l.procedures[req.Procedure]; !ok {
		return handler.Handle(ctx, req, resw)
	}

	mu.Lock()
	if _, ok := l.decorators[req.Procedure]; !ok {
		l.decorators[req.Procedure] = decorator.BuildUnary(decorator.UnaryHandlerWrap(handler), l.procedures[req.Procedure]...)
	}
	mu.Unlock()
	return l.decorators[req.Procedure].Handle(ctx, req, resw)
}

// TransportOnewayMiddleware keeps all the decorator layers defined in the configuration
type TransportOnewayMiddleware struct {
	procedures map[string][]decorator.OnewayDecorator
	decorators map[string]transport.OnewayHandler
}

// HandleOneway all decorators for Oneway transport
func (l TransportOnewayMiddleware) HandleOneway(
	ctx context.Context,
	req *transport.Request,
	handler transport.OnewayHandler,
) error {
	if _, ok := l.procedures[req.Procedure]; !ok {
		return handler.HandleOneway(ctx, req)
	}

	mu.Lock()
	if _, ok := l.decorators[req.Procedure]; !ok {
		l.decorators[req.Procedure] = decorator.BuildOneway(decorator.OnewayHandlerWrap(handler), l.procedures[req.Procedure]...)
	}
	mu.Unlock()
	return l.decorators[req.Procedure].HandleOneway(ctx, req)
}

type authInboundMiddleware struct {
	service.Host
	statsClient *statsClient
}

func (a authInboundMiddleware) Handle(
	ctx context.Context,
	req *transport.Request,
	resw transport.ResponseWriter,
	handler transport.UnaryHandler,
) error {
	fxctx, err := authorize(ctx, a.Host, a.statsClient)
	if err != nil {
		return err
	}
	return handler.Handle(fxctx, req, resw)
}

type authOnewayInboundMiddleware struct {
	service.Host
	statsClient *statsClient
}

func (a authOnewayInboundMiddleware) HandleOneway(
	ctx context.Context,
	req *transport.Request,
	handler transport.OnewayHandler,
) error {
	fxctx, err := authorize(ctx, a.Host, a.statsClient)
	if err != nil {
		return err
	}
	return handler.HandleOneway(fxctx, req)
}

func authorize(ctx context.Context, host service.Host, statsClient *statsClient) (context.Context, error) {
	if err := host.AuthClient().Authorize(ctx); err != nil {
		statsClient.RPCAuthFailCounter().Inc(1)
		ulog.Logger(ctx).Error(auth.ErrAuthorization, zap.Error(err))
		// TODO(anup): GFM-255 update returned error to transport.BadRequestError (user error than server error)
		// https://github.com/yarpc/yarpc-go/issues/687
		return nil, err
	}
	return ctx, nil
}

type panicInboundMiddleware struct {
	statsClient *statsClient
}

func (p panicInboundMiddleware) Handle(
	ctx context.Context,
	req *transport.Request,
	resw transport.ResponseWriter,
	handler transport.UnaryHandler,
) error {
	defer panicRecovery(ctx, p.statsClient)
	return handler.Handle(ctx, req, resw)
}

type panicOnewayInboundMiddleware struct {
	statsClient *statsClient
}

func (p panicOnewayInboundMiddleware) HandleOneway(
	ctx context.Context,
	req *transport.Request,
	handler transport.OnewayHandler,
) error {
	defer panicRecovery(ctx, p.statsClient)
	return handler.HandleOneway(ctx, req)
}

func panicRecovery(ctx context.Context, statsClient *statsClient) {
	if err := recover(); err != nil {
		statsClient.RPCPanicCounter().Inc(1)
		ulog.Logger(ctx).Error("Panic recovered serving request",
			zap.Error(errors.Errorf("panic in handler: %+v", err)),
		)
		// rethrow panic back to yarpc
		// before https://github.com/yarpc/yarpc-go/issues/734 fixed, throw a generic error.
		panic(_panicResponse)
	}
}
