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
	"fmt"
	"net/http"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const _panicResponse = "Server Error"

func tracingInbound(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		operationName := r.Method
		carrier := opentracing.HTTPHeadersCarrier(r.Header)
		spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			ulog.Logger(ctx).Warn("Malformed inbound tracing context: ", zap.Error(err))
		}
		span := opentracing.GlobalTracer().StartSpan(operationName, ext.RPCServerOption(spanCtx))
		ext.HTTPUrl.Set(span, r.URL.String())
		defer span.Finish()

		ctx = opentracing.ContextWithSpan(ctx, span)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func authorizationInbound(next http.Handler, authClient auth.Client, statsClient *statsClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authClient.Authorize(r.Context()); err != nil {
			statsClient.HTTPAuthFailCounter().Inc(1)
			ulog.Logger(r.Context()).Error(auth.ErrAuthorization, zap.Error(err))
			http.Error(w, fmt.Sprintf("Unauthorized access: %+v", err), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// panicInbound handles any panics and return an error
// panic inbound middleware should be added at the end of middleware chain to catch panics
func panicInbound(next http.Handler, statsClient *statsClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func() {
			if err := recover(); err != nil {
				ulog.Logger(ctx).Error("Panic recovered serving request",
					zap.Error(errors.Errorf("panic in handler: %+v", err)),
					zap.Stringer("url", r.URL),
				)
				statsClient.HTTPPanicCounter().Inc(1)
				http.Error(w, _panicResponse, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	}
}

// metricsInbound adds any default metrics related to HTTP
func metricsInbound(next http.Handler, statsClient *statsClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stopwatch := statsClient.HTTPMethodTimer().Timer(r.Method).Start()
		defer stopwatch.Stop()
		defer statsClient.HTTPStatusCountScope().Tagged(map[string]string{_tagStatus: w.Header().Get("Status")}).Counter("total").Inc(1)
		next.ServeHTTP(w, r)
	}
}
