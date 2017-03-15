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

package ulog

import (
	"context"
	"testing"

	"go.uber.org/fx/testutils"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestLogger(t *testing.T) {
	t.Run("Nil context", func(t *testing.T) {
		assert.Equal(t, zap.L(), Logger(nil), "Expected to fall back to global Logger.")
		assert.Equal(t, zap.S(), Sugar(nil), "Expected to fall back to global SugaredLogger.")
	})

	t.Run("Non-nil context", func(t *testing.T) {
		testutils.WithInMemoryLogger(t, nil, func(logger *zap.Logger, buf *zaptest.Buffer) {
			defer zap.ReplaceGlobals(logger)()
			withTracedContext(func(ctx context.Context) {
				Logger(ctx).Info("")
				Sugar(ctx).Info("")
			})
			output := buf.Lines()
			assert.Equal(t, 2, len(output))
			for _, line := range output {
				assert.Regexp(
					t,
					`{"level":"info","msg":"","trace":{"span":\"[a-zA-Z0-9]+\","trace":\"[a-zA-Z0-9]+\"}}`,
					line,
					"Expected output to contain tracing info.",
				)
			}
		})
	})
}

func TestTraceField(t *testing.T) {
	assert.Equal(t, zap.Skip(), Trace(nil), "Expected Trace of a nil context to be a no-op.")

	withTracedContext(func(ctx context.Context) {
		enc := zapcore.NewMapObjectEncoder()
		Trace(ctx).AddTo(enc)

		logged, ok := enc.Fields["trace"].(map[string]interface{})
		require.True(t, ok, "Expected trace to be a map.")
		keys := make(map[string]struct{}, len(logged))
		for k := range logged {
			keys[k] = struct{}{}
		}

		// We could extract the span from the context and assert specific IDs,
		// but that just copies the production code.
		assert.Equal(
			t,
			map[string]struct{}{"span": {}, "trace": {}},
			keys,
			"Expected to log span and trace IDs.",
		)
	})
}

func withTracedContext(f func(ctx context.Context)) {
	tracer, closer := jaeger.NewTracer(
		"serviceName", jaeger.NewConstSampler(true), jaeger.NewNullReporter(),
	)
	defer closer.Close()

	ctx := opentracing.ContextWithSpan(context.Background(), tracer.StartSpan("test"))
	f(ctx)
}
