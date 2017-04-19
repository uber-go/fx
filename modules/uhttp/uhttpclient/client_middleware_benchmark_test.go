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

package uhttpclient

import (
	"context"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/tracing"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

// BenchmarkClientMiddleware/empty-8          100000000           10.8 ns/op         0 B/op          0 allocs/op
// BenchmarkClientMiddleware/tracing-8          500000          3918 ns/op        1729 B/op         27 allocs/op
// BenchmarkClientMiddleware/auth-8            1000000          1866 ns/op         719 B/op         14 allocs/op
// BenchmarkClientMiddleware/default-8          300000          5604 ns/op        2477 B/op         41 allocs/op
func BenchmarkClientMiddleware(b *testing.B) {
	tracer, closer, err := tracing.InitGlobalTracer(
		&jconfig.Configuration{}, "Test", zap.NewNop(), tally.NoopScope,
	)
	if err != nil {
		b.Error(err)
	}

	defer closer.Close()
	cfg := config.NewYAMLProviderFromBytes(_testYaml)
	bm := map[string][]OutboundMiddleware{
		"empty":   {},
		"tracing": {tracingOutbound(tracer)},
		"auth":    {authenticationOutbound(cfg, auth.Load(cfg, tally.NoopScope))},
		"default": {tracingOutbound(tracer), authenticationOutbound(cfg, auth.Load(cfg, tally.NoopScope))},
	}

	for name, middleware := range bm {
		chain := newExecutionChain(middleware, nopTransport{})
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
