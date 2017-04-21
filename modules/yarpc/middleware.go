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

	"go.uber.org/fx/auth"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"

	"go.uber.org/yarpc/api/transport"
)

const _panicResponse = "Server Error"

type authInboundMiddleware struct {
	authClient auth.Client
}

func (a authInboundMiddleware) Handle(
	ctx context.Context,
	req *transport.Request,
	resw transport.ResponseWriter,
	handler transport.UnaryHandler,
) error {
	fxctx, err := authorize(ctx, a.authClient)
	if err != nil {
		return err
	}
	return handler.Handle(fxctx, req, resw)
}

type authOnewayInboundMiddleware struct {
	authClient auth.Client
}

func (a authOnewayInboundMiddleware) HandleOneway(
	ctx context.Context,
	req *transport.Request,
	handler transport.OnewayHandler,
) error {
	fxctx, err := authorize(ctx, a.authClient)
	if err != nil {
		return err
	}
	return handler.HandleOneway(fxctx, req)
}

func authorize(ctx context.Context, authClient auth.Client) (context.Context, error) {
	if err := authClient.Authorize(ctx); err != nil {
		ulog.Logger(ctx).Error(auth.ErrAuthorization, zap.Error(err))
		// TODO(anup): GFM-255 update returned error to transport.BadRequestError (user error than server error)
		// https://github.com/yarpc/yarpc-go/issues/687
		return nil, err
	}
	return ctx, nil
}
