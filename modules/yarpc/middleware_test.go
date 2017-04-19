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
	"errors"
	"testing"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/ulog"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func checkLogForTrace(t *testing.T, buf *zaptest.Buffer) {
	require.True(t, len(buf.Lines()) > 0)
	for _, line := range buf.Lines() {
		assert.Contains(t, line, "trace")
		assert.Contains(t, line, "span")
	}

}

func TestInboundMiddleware_auth(t *testing.T) {
	unary := authInboundMiddleware{auth.NopClient}
	err := unary.Handle(context.Background(), &transport.Request{}, nil, &fakeUnary{t: t})
	assert.EqualError(t, err, "handle")
}

func TestInboundMiddleware_authFailure(t *testing.T) {
	unary := authInboundMiddleware{auth.FailureClient}
	err := unary.Handle(context.Background(), &transport.Request{}, nil, &fakeUnary{t: t})
	assert.EqualError(t, err, "Error authorizing the service")

}

func TestOnewayInboundMiddleware_auth(t *testing.T) {
	oneway := authOnewayInboundMiddleware{auth.NopClient}
	err := oneway.HandleOneway(context.Background(), &transport.Request{}, &fakeOneway{t: t})
	assert.EqualError(t, err, "oneway handle")
}

func TestOnewayInboundMiddleware_authFailure(t *testing.T) {
	oneway := authOnewayInboundMiddleware{auth.FailureClient}
	err := oneway.HandleOneway(context.Background(), &transport.Request{}, &fakeOneway{t: t})
	assert.EqualError(t, err, "Error authorizing the service")
}

type fakeEnveloper struct {
	serviceName string
}

func (f fakeEnveloper) MethodName() string {
	return f.serviceName
}

func (f fakeEnveloper) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

func (f fakeEnveloper) ToWire() (wire.Value, error) {
	return wire.NewValueStruct(wire.Struct{}), nil
}

type fakeUnary struct {
	t *testing.T
}

func (f fakeUnary) Handle(
	ctx context.Context,
	_param1 *transport.Request,
	_param2 transport.ResponseWriter,
) error {
	ulog.Logger(ctx).Info("fakeUnary")
	assert.NotNil(f.t, ctx)
	return errors.New("handle")
}

type fakeOneway struct {
	t *testing.T
}

func (f fakeOneway) HandleOneway(ctx context.Context, p *transport.Request) error {
	ulog.Logger(ctx).Info("fakeOneway")
	assert.NotNil(f.t, ctx)
	return errors.New("oneway handle")
}

type alwaysPanicUnary struct{}

func (p alwaysPanicUnary) Handle(_ context.Context, _ *transport.Request, _ transport.ResponseWriter) error {
	panic("panic")
}

type alwaysPanicOneway struct{}

func (p alwaysPanicOneway) HandleOneway(_ context.Context, _ *transport.Request) error {
	panic("panic")
}
