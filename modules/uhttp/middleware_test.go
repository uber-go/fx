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

	"go.uber.org/fx/auth"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestDefaultInboundMiddlewareWithNopHost(t *testing.T) {
	tests := []struct {
		desc   string
		testFn func(*testing.T, tally.Scope)
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

	t.Run("parallel group", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt // capture range variable
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				tt.testFn(t, tally.NoopScope)
			})
		}
	})
}

func TestDefaultMiddlewareWithNopHostAuthFailure(t *testing.T) {
	tests := []struct {
		desc   string
		testFn func(*testing.T, tally.Scope)
	}{
		{
			desc:   "testInboundMiddlewareChainAuthFailure",
			testFn: testInboundMiddlewareChainAuthFailure,
		},
	}

	t.Run("parallel group", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt // capture range variable
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				tt.testFn(t, tally.NoopScope)
			})
		}
	})
}

func TestDefaultInboundMiddlewareWithNopHostConfigured(t *testing.T) {
	// this test's sub tests cannot run parallel
	// and they need to build host by themselves
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
	}
}

func testInboundMiddlewareChain(t *testing.T, s tally.Scope) {
	response := testServeHTTP(getNopHandler())
	assert.True(t, strings.Contains(response.Body.String(), "inbound middleware ok"))
}

func testTracingInboundWithLogs(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(loggerWithZap *zap.Logger, buf *zaptest.Buffer) {
		defer ulog.SetLogger(loggerWithZap)()
		// Create a Jaeger tracer.
		jConfig := &config.Configuration{
			Sampler:  &config.SamplerConfig{Type: "const", Param: 1.0},
			Reporter: &config.ReporterConfig{LogSpans: true},
		}
		tracer, closer, err := tracing.InitGlobalTracer(
			jConfig, "serviceName", loggerWithZap, tally.NoopScope,
		)
		assert.NoError(t, err)
		defer closer.Close()
		opentracing.InitGlobalTracer(tracer)
		defer opentracing.InitGlobalTracer(opentracing.NoopTracer{})
		response := testServeHTTP(tracingInbound(getNopHandler()))

		assert.Contains(t, response.Body.String(), "inbound middleware ok")
		assert.True(t, len(buf.Lines()) > 0)
		tracecount := 0
		spancount := 0
		t.Log(buf.Lines())
		for _, line := range buf.Lines() {
			if strings.Contains(line, `"trace":`) {
				tracecount++
			}
			if strings.Contains(line, `"span":`) {
				spancount++
			}
		}
		assert.Equal(t, tracecount, 1)
		assert.Equal(t, spancount, 1)
	})
}

func testInboundTraceInboundAuthChain(t *testing.T, s tally.Scope) {
	response := testServeHTTP(authorizationInbound(tracingInbound(getNopHandler()), auth.NopClient, newStatsClient(s)))
	assert.Contains(t, response.Body.String(), "inbound middleware ok")
}

func testInboundMiddlewareChainAuthFailure(t *testing.T, s tally.Scope) {
	response := testServeHTTP(authorizationInbound(tracingInbound(getNopHandler()), auth.FailureClient, newStatsClient(s)))
	assert.Equal(t, response.Body.String(), "Unauthorized access: Error authorizing the service\n")
	assert.Equal(t, 401, response.Code)
}

func testPanicInbound(t *testing.T, s tally.Scope) {
	response := testServeHTTP(panicInbound(getPanicHandler(), newStatsClient(s)))
	assert.Equal(t, response.Body.String(), _panicResponse+"\n")
	assert.Equal(t, http.StatusInternalServerError, response.Code)

	snapshot := s.(tally.TestScope).Snapshot()
	counters := snapshot.Counters()
	assert.True(t, counters["panic"].Value() > 0)
}

func testMetricsInbound(t *testing.T, s tally.Scope) {
	response := testServeHTTP(metricsInbound(getNopHandler(), newStatsClient(s)))
	assert.Contains(t, response.Body.String(), "inbound middleware ok")

	snapshot := s.(tally.TestScope).Snapshot()
	counters := snapshot.Counters()
	timers := snapshot.Timers()
	assert.True(t, counters["total"].Value() > 0)
	assert.NotNil(t, timers["GET"].Values())
}

func testServeHTTP(chain http.Handler) *httptest.ResponseRecorder {
	request := httptest.NewRequest("", "http://middleware", nil)
	response := httptest.NewRecorder()
	chain.ServeHTTP(response, request)
	return response
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
