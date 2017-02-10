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

package uhttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/modules/uhttp/stats"
	"go.uber.org/fx/service"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/zap"
	"github.com/uber/jaeger-client-go/config"
)

func TestFilterChain(t *testing.T) {
	host := service.NopHost()
	chain := newFilterChainBuilder(host).AddFilters([]Filter{}...).Build(getNopHandler(host))
	response := testServeHTTP(chain, host)
	assert.True(t, strings.Contains(response.Body.String(), "filters ok"))
}

func TestTracingFilterWithLogs(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		// Create in-memory logger and jaeger tracer
		loggerWithZap := ulog.Builder().SetLogger(zapLogger).Build()
		jConfig := &config.Configuration{
			Sampler:  &config.SamplerConfig{Type: "const", Param: 1.0},
			Reporter: &config.ReporterConfig{LogSpans: true},
		}
		tracer, closer, err := tracing.InitGlobalTracer(
			jConfig, "serviceName", loggerWithZap, metrics.NopCachedStatsReporter,
		)
		assert.NoError(t, err)
		defer closer.Close()
		opentracing.InitGlobalTracer(tracer)
		defer opentracing.InitGlobalTracer(opentracing.NoopTracer{})

		host := service.NopHostConfigured(auth.NopClient, loggerWithZap, tracer)
		chain := newFilterChainBuilder(host).AddFilters([]Filter{contextFilter{host}, tracingServerFilter{}}...).Build(getNopHandler(host))
		response := testServeHTTP(chain, host)
		assert.Contains(t, response.Body.String(), "filters ok")
		assert.True(t, len(buf.Lines()) > 0)
		var tracecount = 0
		var spancount = 0
		for _, line := range buf.Lines() {
			if strings.Contains(line, "traceID") {
				tracecount++
			}
			if strings.Contains(line, "spanID") {
				spancount++
			}
		}
		assert.Equal(t, tracecount, 1)
		assert.Equal(t, spancount, 1)
	})
}

func TestFilterChainFilters(t *testing.T) {
	host := service.NopHost()
	chain := newFilterChainBuilder(host).AddFilters(
		tracingServerFilter{},
		authorizationFilter{
			authClient: host.AuthClient(),
		}).Build(getNopHandler(host))

	response := testServeHTTP(chain, host)
	assert.Contains(t, response.Body.String(), "filters ok")
}

func TestFilterChainFilters_AuthFailure(t *testing.T) {
	host := service.NopHostAuthFailure()
	stats.SetupHTTPMetrics(host.Metrics())
	chain := newFilterChainBuilder(host).AddFilters(
		tracingServerFilter{},
		authorizationFilter{
			authClient: host.AuthClient(),
		}).Build(getNopHandler(host))
	response := testServeHTTP(chain, host)
	assert.Contains(t, "Unauthorized access: Error authorizing the service", response.Body.String())
	assert.Equal(t, 401, response.Code)
}

func testServeHTTP(chain filterChain, host service.Host) *httptest.ResponseRecorder {
	request := httptest.NewRequest("", "http://filters", nil)
	response := httptest.NewRecorder()
	ctx := context.Background()
	chain.ServeHTTP(ctx, response, request)
	return response
}

func getNopHandler(host service.Host) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		fx.Logger(ctx).Info("Inside Noop Handler")
		io.WriteString(w, "filters ok")
	}
}
