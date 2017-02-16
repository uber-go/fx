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

package client

import (
	"context"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
)

// BenchmarkClientMiddlewares/empty-8          100000000           10.8 ns/op         0 B/op          0 allocs/op
// BenchmarkClientMiddlewares/tracing-8          500000          3918 ns/op        1729 B/op         27 allocs/op
// BenchmarkClientMiddlewares/auth-8            1000000          1866 ns/op         719 B/op         14 allocs/op
// BenchmarkClientMiddlewares/default-8          300000          5604 ns/op        2477 B/op         41 allocs/op
func BenchmarkClientMiddlewares(b *testing.B) {
	tracer, closer, err := tracing.InitGlobalTracer(&jconfig.Configuration{}, "Test", ulog.NopLogger, metrics.NopCachedStatsReporter)
	if err != nil {
		b.Error(err)
	}

	defer closer.Close()
	bm := map[string][]OutboundMiddleware{
		"empty":   {},
		"tracing": {tracingOutbound()},
		"auth":    {authenticationOutbound(fakeAuthInfo{_testYaml})},
		"default": {tracingOutbound(), authenticationOutbound(fakeAuthInfo{_testYaml})},
	}

	for name, middlewares := range bm {
		chain := newExecutionChain(middlewares, nopTransport{})
		span := tracer.StartSpan("test_method")
		span.SetBaggageItem(auth.ServiceAuth, "testService")

		ctx := opentracing.ContextWithSpan(context.Background(), span)

		req := httptest.NewRequest("", "http://localhost", nil).WithContext(ctx)

		b.ResetTimer()
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := chain.RoundTrip(req); err != nil {
					b.Error(err)
				}
			}
		})
	}
}
