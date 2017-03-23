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

	"go.uber.org/fx/config"
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/testutils/tracing"
	"go.uber.org/fx/ulog"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestInboundMiddleware_Context(t *testing.T) {
	host := service.NopHost()
	unary := contextInboundMiddleware{newStatsClient(host.Metrics())}
	testutils.WithInMemoryLogger(t, nil, func(loggerWithZap *zap.Logger, buf *zaptest.Buffer) {
		defer ulog.SetLogger(loggerWithZap)()
		tracing.WithSpan(t, loggerWithZap, func(span opentracing.Span) {
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			err := unary.Handle(ctx, &transport.Request{}, nil, &fakeUnary{t: t})
			require.Contains(t, err.Error(), "handle")
			checkLogForTrace(t, buf)
		})
	})
}

func TestOnewayInboundMiddleware_Context(t *testing.T) {
	oneway := contextOnewayInboundMiddleware{}
	testutils.WithInMemoryLogger(t, nil, func(loggerWithZap *zap.Logger, buf *zaptest.Buffer) {
		defer ulog.SetLogger(loggerWithZap)()
		tracing.WithSpan(t, loggerWithZap, func(span opentracing.Span) {
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			err := oneway.HandleOneway(ctx, &transport.Request{}, &fakeOneway{t: t})
			require.Contains(t, err.Error(), "oneway handle")
			checkLogForTrace(t, buf)
		})
	})
}

func checkLogForTrace(t *testing.T, buf *zaptest.Buffer) {
	require.True(t, len(buf.Lines()) > 0)
	for _, line := range buf.Lines() {
		assert.Contains(t, line, "trace")
		assert.Contains(t, line, "span")
	}

}

func TestInboundMiddleware_auth(t *testing.T) {
	host := service.NopHost()
	unary := authInboundMiddleware{host, newStatsClient(host.Metrics())}
	err := unary.Handle(context.Background(), &transport.Request{}, nil, &fakeUnary{t: t})
	assert.EqualError(t, err, "handle")
}

func TestInboundMiddleware_authFailure(t *testing.T) {
	host := service.NopHostAuthFailure()
	unary := authInboundMiddleware{host, newStatsClient(host.Metrics())}
	err := unary.Handle(context.Background(), &transport.Request{}, nil, &fakeUnary{t: t})
	assert.EqualError(t, err, "Error authorizing the service")

}

func TestOnewayInboundMiddleware_auth(t *testing.T) {
	oneway := authOnewayInboundMiddleware{
		Host: service.NopHost(),
	}
	err := oneway.HandleOneway(context.Background(), &transport.Request{}, &fakeOneway{t: t})
	assert.EqualError(t, err, "oneway handle")
}

func TestOnewayInboundMiddleware_authFailure(t *testing.T) {
	host := service.NopHostAuthFailure()
	oneway := authOnewayInboundMiddleware{host, newStatsClient(host.Metrics())}
	err := oneway.HandleOneway(context.Background(), &transport.Request{}, &fakeOneway{t: t})
	assert.EqualError(t, err, "Error authorizing the service")
}

func TestInboundMiddleware_panic(t *testing.T) {
	host := service.NopHost()
	testScope := host.Metrics()
	statsClient := newStatsClient(testScope)

	defer testPanicHandler(t, testScope)
	unary := panicInboundMiddleware{statsClient}
	unary.Handle(context.Background(), &transport.Request{}, nil, &alwaysPanicUnary{})
}

func TestInboundMiddleware_TransportUnaryMiddleware(t *testing.T) {

	host := service.NopHost()

	m := TransportUnaryMiddleware{
		procedureMap: make(map[string][]decorator.Decorator),
		layerMap:     make(map[string]transport.UnaryHandler),
	}
	decorator := decorator.Recovery(host.Metrics(), config.NewScopedProvider("recovery", host.Config()))
	m.procedureMap["hello"] = append(m.procedureMap["recovery"], decorator)
	m.Handle(context.Background(), &transport.Request{
		Procedure: "hello",
	}, nil, &fakeUnary{t: t})
	m.Handle(context.Background(), &transport.Request{
		Procedure: "hello",
	}, nil, &fakeUnary{t: t})
	m.Handle(context.Background(), &transport.Request{
		Procedure: "hello",
	}, nil, &fakeUnary{t: t})
}

func TestOnewayInboundMiddleware_panic(t *testing.T) {
	host := service.NopHost()
	testScope := host.Metrics()
	statsClient := newStatsClient(testScope)

	defer testPanicHandler(t, testScope)
	oneway := panicOnewayInboundMiddleware{statsClient}
	oneway.HandleOneway(context.Background(), &transport.Request{}, &alwaysPanicOneway{})
}

func testPanicHandler(t *testing.T, testScope tally.Scope) {
	r := recover()
	assert.EqualValues(t, r, _panicResponse)

	snapshot := testScope.(tally.TestScope).Snapshot()
	counters := snapshot.Counters()
	assert.True(t, counters["panic"].Value() > 0)
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
