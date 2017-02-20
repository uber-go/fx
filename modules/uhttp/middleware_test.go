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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/fx/metrics"
	"go.uber.org/fx/modules/uhttp/internal/stats"
	"go.uber.org/fx/service"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"github.com/uber-go/zap"
	"github.com/uber/jaeger-client-go/config"
)

func TestDefaultInboundMiddlewareWithNopHost(t *testing.T) {
	tests := []struct {
		desc   string
		testFn func(*testing.T, service.Host)
	}{
		{
			desc:   "testInboundMiddlewareChain",
			testFn: testInboundMiddlewareChain,
		},
		{
			desc:   "testInboundTraceInboundAuthChain",
			testFn: testInboundTraceInboundAuthChain,
		},
		{
			desc:   "testPanicInbound",
			testFn: testPanicInbound,
		},
		{
			desc:   "testMetricsInbound",
			testFn: testMetricsInbound,
		},
	}

	// setup
	host := service.NopHost()
	stats.SetupHTTPMetrics(host.Metrics())
	// teardown
	defer httpMetricsTeardown()

	t.Run("parallel group", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt // capture range variable
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				tt.testFn(t, host)
			})
		}
	})
}

func TestDefaultMiddlewareWithNopHostAuthFailure(t *testing.T) {
	tests := []struct {
		desc   string
		testFn func(*testing.T, service.Host)
	}{
		{
			desc:   "testInboundMiddlewareChainAuthFailure",
			testFn: testInboundMiddlewareChainAuthFailure,
		},
	}

	// setup
	host := service.NopHostAuthFailure()
	stats.SetupHTTPMetrics(host.Metrics())
	// teardown
	defer httpMetricsTeardown()

	t.Run("parallel group", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt // capture range variable
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				tt.testFn(t, host)
			})
		}
	})
}

func TestDefaultInboundMiddlewareWithNopHostConfigured(t *testing.T) {
	// this test's sub tests cannot run parallel
	// and they need to build host by theirselves
	tests := []struct {
		desc   string
		testFn func(*testing.T)
	}{
		{
			desc:   "testTracingInboundWithLogs",
			testFn: testTracingInboundWithLogs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, tt.testFn)
		httpMetricsTeardown()
	}
}

func testInboundMiddlewareChain(t *testing.T, host service.Host) {
	chain := newInboundMiddlewareChainBuilder().AddMiddleware([]InboundMiddleware{}...).Build(getNopHandler())
	response := testServeHTTP(chain)
	assert.True(t, strings.Contains(response.Body.String(), "inbound middleware ok"))
}

func testTracingInboundWithLogs(t *testing.T) {
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

		ulog.SetLogger(loggerWithZap)
		chain := newInboundMiddlewareChainBuilder().AddMiddleware([]InboundMiddleware{contextInbound{loggerWithZap}, tracingInbound{}}...).Build(getNopHandler())
		response := testServeHTTP(chain)
		assert.Contains(t, response.Body.String(), "inbound middleware ok")
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

func testInboundTraceInboundAuthChain(t *testing.T, host service.Host) {
	chain := newInboundMiddlewareChainBuilder().AddMiddleware(
		tracingInbound{},
		authorizationInbound{
			authClient: host.AuthClient(),
		}).Build(getNopHandler())

	response := testServeHTTP(chain)
	assert.Contains(t, response.Body.String(), "inbound middleware ok")
}

func testInboundMiddlewareChainAuthFailure(t *testing.T, host service.Host) {
	chain := newInboundMiddlewareChainBuilder().AddMiddleware(
		tracingInbound{},
		authorizationInbound{
			authClient: host.AuthClient(),
		}).Build(getNopHandler())
	response := testServeHTTP(chain)
	assert.Equal(t, response.Body.String(), "Unauthorized access: Error authorizing the service\n")
	assert.Equal(t, 401, response.Code)
}

func testPanicInbound(t *testing.T, host service.Host) {
	chain := newInboundMiddlewareChainBuilder().AddMiddleware(
		panicInbound{},
	).Build(getPanicHandler())
	response := testServeHTTP(chain)
	assert.Equal(t, response.Body.String(), _panicResponse+"\n")
	assert.Equal(t, http.StatusInternalServerError, response.Code)

	testScope := host.Metrics()
	snapshot := testScope.(tally.TestScope).Snapshot()
	counters := snapshot.Counters()
	assert.True(t, counters["panic"].Value() > 0)
}

func testMetricsInbound(t *testing.T, host service.Host) {
	chain := newInboundMiddlewareChainBuilder().AddMiddleware(
		metricsInbound{},
	).Build(getNopHandler())
	response := testServeHTTP(chain)
	assert.Contains(t, response.Body.String(), "inbound middleware ok")

	testScope := host.Metrics()
	snapshot := testScope.(tally.TestScope).Snapshot()
	counters := snapshot.Counters()
	timers := snapshot.Timers()
	assert.True(t, counters["total"].Value() > 0)
	assert.NotNil(t, timers["GET"].Values())
}

func testServeHTTP(chain inboundMiddlewareChain) *httptest.ResponseRecorder {
	request := httptest.NewRequest("", "http://middleware", nil)
	response := httptest.NewRecorder()
	chain.ServeHTTP(response, request)
	return response
}

func httpMetricsTeardown() {
	stats.HTTPPanicCounter = nil
	stats.HTTPAuthFailCounter = nil
	stats.HTTPMethodTimer = nil
	stats.HTTPStatusCountScope = nil
}

func getNopHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ulog.Logger(r.Context()).Info("Inside Noop Handler")
		io.WriteString(w, "inbound middleware ok")
	}
}

func getPanicHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		panic("panic")
	}
}
