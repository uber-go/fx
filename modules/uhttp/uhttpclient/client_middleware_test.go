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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/tracing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

var (
	_respOK   = &http.Response{StatusCode: http.StatusOK}
	_req      = func() *http.Request { return httptest.NewRequest("", "http://localhost", nil) }
	errClient = errors.New("client test error")
)

func TestExecutionChain(t *testing.T) {
	t.Parallel()
	execChain := newExecutionChain([]OutboundMiddleware{}, nopTransport{})
	resp, err := execChain.RoundTrip(_req().WithContext(context.Background()))
	assert.NoError(t, err)
	assert.Equal(t, _respOK, resp)
}

func TestExecutionChainOutboundMiddleware(t *testing.T) {
	execChain := newExecutionChain(
		[]OutboundMiddleware{tracingOutbound(opentracing.NoopTracer{})}, nopTransport{},
	)
	resp, err := execChain.RoundTrip(_req())
	assert.NoError(t, err)
	assert.Equal(t, _respOK, resp)
}

func TestExecutionChainOutboundMiddlewareError(t *testing.T) {
	execChain := newExecutionChain(
		[]OutboundMiddleware{tracingOutbound(opentracing.NoopTracer{})}, errTransport{},
	)
	resp, err := execChain.RoundTrip(_req().WithContext(context.Background()))
	assert.Error(t, err)
	assert.Equal(t, errClient, err)
	assert.Nil(t, resp)
}

func withOpentracingSetup(
	t *testing.T,
	registerFunc auth.RegisterFunc,
	fn func(tracer opentracing.Tracer),
) {
	tracer, closer, err := tracing.CreateTracer(
		&jconfig.Configuration{}, "Test", zap.NewNop(), tally.NoopScope,
	)

	defer closer.Close()
	assert.NotNil(t, closer)
	require.NoError(t, err)

	auth.UnregisterClient()
	defer auth.UnregisterClient()
	auth.RegisterClient(registerFunc)
	fn(tracer)
}

func TestExecutionChainOutboundMiddleware_AuthContextPropagation(t *testing.T) {
	withOpentracingSetup(t, nil, func(tracer opentracing.Tracer) {
		execChain := getExecChainWithAuth(t)
		span := tracer.StartSpan("test_method")
		span.SetBaggageItem(auth.ServiceAuth, "test_service")
		ctx := opentracing.ContextWithSpan(context.Background(), span)

		resp, err := execChain.RoundTrip(_req().WithContext(ctx))
		assert.NoError(t, err)
		assert.Equal(t, _respOK, resp)
	})
}

func TestExecutionChainOutboundMiddleware_AuthContextPropagationFailure(t *testing.T) {
	withOpentracingSetup(t, auth.FakeFailureClient, func(tracer opentracing.Tracer) {
		execChain := getExecChainWithAuth(t)
		span := tracer.StartSpan("test_method")
		span.SetBaggageItem(auth.ServiceAuth, "testService")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		resp, err := execChain.RoundTrip(_req().WithContext(ctx))
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func getExecChainWithAuth(t *testing.T) executionChain {
	return newExecutionChain(
		[]OutboundMiddleware{authenticationOutbound(fakeAuthInfo{_testYaml})},
		contextPropagationTransport{t},
	)
}

func TestOutboundMiddlewareWithTracerErrors(t *testing.T) {
	testCases := map[string]OutboundMiddleware{
		"auth":    authenticationOutbound(fakeAuthInfo{_testYaml}),
		"tracing": tracingOutbound(opentracing.NoopTracer{}),
	}

	for name, middleware := range testCases {
		op := func() {
			tr := &shadowTracer{
				opentracing.NoopTracer{},
				func(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
					return errors.New("Very bad tracer")
				},
				nil,
			}

			execChain := newExecutionChain(
				[]OutboundMiddleware{middleware, tracingOutbound(tr)}, nopTransport{})
			span := tr.Tracer.StartSpan("test_method")
			span.SetBaggageItem(auth.ServiceAuth, "testService")
			sp := &shadowSpan{span, tr}
			tr.span = sp

			ctx := opentracing.ContextWithSpan(context.Background(), sp)

			_, err := execChain.RoundTrip(_req().WithContext(ctx))
			assert.EqualError(t, err, "Very bad tracer")
		}

		t.Run(name, func(t *testing.T) { op() })
	}
}

type fakeAuthInfo struct {
	yaml []byte
}

func (f fakeAuthInfo) Config() config.Provider {
	return config.NewYAMLProviderFromBytes(f.yaml)
}

func (f fakeAuthInfo) Logger() *zap.Logger {
	return zap.NewNop()
}

func (f fakeAuthInfo) Metrics() tally.Scope {
	return tally.NoopScope
}

type contextPropagationTransport struct {
	*testing.T
}

func (tr contextPropagationTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ctx := req.Context()

	span := opentracing.SpanFromContext(ctx)
	assert.NotNil(tr.T, span)
	assert.Equal(tr.T, "test_service", span.BaggageItem(auth.ServiceAuth))
	return _respOK, nil
}

type nopTransport struct{}

func (tr nopTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return _respOK, nil
}

type errTransport struct{}

func (tr errTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return nil, errClient
}

type shadowTracer struct {
	opentracing.Tracer
	inject func(sm opentracing.SpanContext, format interface{}, carrier interface{}) error
	span   opentracing.Span
}

func (s *shadowTracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return s.inject(sm, format, carrier)
}

func (s *shadowTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return s.span
}

type shadowSpan struct {
	opentracing.Span
	tracer opentracing.Tracer
}

func (s *shadowSpan) Tracer() opentracing.Tracer {
	return s.tracer
}
