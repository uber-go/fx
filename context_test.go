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

package fx

import (
	"context"
	"testing"

	"go.uber.org/fx/service"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	ztestutils "go.uber.org/zap/testutils"
)

type testKey int

const (
	_testContextKey testKey = iota
	_testContextKey2
)

func TestContext_LoggerAccess(t *testing.T) {
	ctx := NewContext(context.Background(), service.NopHost())
	assert.NotNil(t, ctx)
	assert.NotNil(t, Logger(ctx))
	assert.NotNil(t, ctx.Value(_contextHost))
}

func TestWithContextAwareLogger(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zapLogger *zap.Logger, buf *ztestutils.Buffer) {
		// Create in-memory logger and jaeger tracer
		loggerWithZap, err := ulog.Builder().SetLogger(zapLogger).Build()
		require.NoError(t, err)
		tracer, closer := jaeger.NewTracer(
			"serviceName", jaeger.NewConstSampler(true), jaeger.NewNullReporter(),
		)
		defer closer.Close()
		span := tracer.StartSpan("opName")
		ctx := context.WithValue(context.Background(), _contextHost, ctxHost{
			log: loggerWithZap,
		})
		ctx = WithContextAwareLogger(ctx, span)
		Logger(ctx).Info("Testing context aware logger")
		assert.True(t, len(buf.Lines()) > 0)
		for _, line := range buf.Lines() {
			assert.Contains(t, line, "traceID")
			assert.Contains(t, line, "spanID")
		}
	})
}

func TestWithContext_NilHost(t *testing.T) {
	ctx := NewContext(context.Background(), nil)
	assert.NotNil(t, Logger(ctx))
}

func TestContext_Convert(t *testing.T) {
	host := service.NopHost()
	ctx := NewContext(context.Background(), host)
	logger := Logger(ctx)
	assert.Equal(t, host.Logger(), logger)

	assertConvert(ctx, t, logger)
}

func assertConvert(ctx context.Context, t *testing.T, logger ulog.Log) {
	assert.NotNil(t, Logger(ctx))
	assert.Equal(t, Logger(ctx), logger)

	ctx = context.WithValue(ctx, _testContextKey, nil)

	assert.NotNil(t, Logger(ctx))
	assert.Equal(t, Logger(ctx), logger)
}
