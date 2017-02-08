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

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

type contextKey int

const _contextHost contextKey = iota

type ctxHost struct {
	log ulog.Log
}

func contextHost(ctx context.Context) ctxHost {
	c := ctx.Value(_contextHost)
	if c == nil {
		c = ctxHost{}
		ctx = context.WithValue(ctx, _contextHost, c)
	}
	return c.(ctxHost)
}

// SetContextHost sets the context with context aware logger
func SetContextHost(ctx context.Context, host service.Host) context.Context {
	if host != nil {
		ctx = context.WithValue(ctx, _contextHost, ctxHost{
			log: host.Logger(),
		})
	}
	return ctx
}

// WithContextAwareLogger returns a new context with a context-aware logger
func WithContextAwareLogger(ctx context.Context, span opentracing.Span) context.Context {
	store := contextHost(ctx)
	// Note that opentracing.Tracer does not expose the tracer id
	// We only inject tracing information for jaeger.Tracer
	if jSpanCtx, ok := span.Context().(jaeger.SpanContext); ok {
		store.log = Logger(ctx).With(
			"traceID", jSpanCtx.TraceID(), "spanID", jSpanCtx.SpanID(),
		)
	}
	return context.WithValue(ctx, _contextHost, store)
}

// Logger returns a context aware logger. If logger is absent from the context store,
// the function updates the context with a new context based logger
func Logger(ctx context.Context) ulog.Log {
	store := contextHost(ctx)
	if store.log == nil {
		store.log = ulog.Logger()
	}
	return store.log
}
