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
	"net/http"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Executor executes the http request. Execute must be safe to use by multiple go routines
type Executor interface {
	Execute(r *http.Request) (resp *http.Response, err error)
}

// OutboundMiddleware applies outbound middlewares on client requests and such as adding tracing to request's context.
// Outbound middlewares must call next.Execute() at most once, calling it twice and more
// will lead to an undefined behavior
type OutboundMiddleware interface {
	Handle(r *http.Request, next Executor) (resp *http.Response, err error)
}

// OutboundMiddlewareFunc is an adaptor to call normal functions to apply outbound middlewares.
type OutboundMiddlewareFunc func(r *http.Request, next Executor) (resp *http.Response, err error)

// Handle implements Handle from the OutboundMiddleware interface and simply delegates to the function
func (f OutboundMiddlewareFunc) Handle(r *http.Request, next Executor) (resp *http.Response, err error) {
	return f(r, next)
}

func tracingOutbound() OutboundMiddlewareFunc {
	return func(req *http.Request, next Executor) (resp *http.Response, err error) {
		ctx := req.Context()
		opName := req.Method
		var parent opentracing.SpanContext
		if s := opentracing.SpanFromContext(ctx); s != nil {
			parent = s.Context()
		}

		// TODO(alsam) This makes our client to be not safe to use by multiple go routines.
		span := opentracing.GlobalTracer().StartSpan(opName, opentracing.ChildOf(parent))
		ext.SpanKindRPCClient.Set(span)
		ext.HTTPUrl.Set(span, req.URL.String())
		defer span.Finish()

		ctx = opentracing.ContextWithSpan(ctx, span)

		if err := injectSpanIntoHeaders(req.Header, span); err != nil {
			return nil, err
		}

		resp, err = next.Execute(req.WithContext(ctx))
		if resp != nil {
			span.SetTag("http.status_code", resp.StatusCode)
		}
		if err != nil {
			span.SetTag("error", err.Error())
		}
		return resp, err
	}
}

// authenticationOutbound on client side calls authenticate, and gets a claim that client is who they say they are
// We only authorize with the claim on server side
func authenticationOutbound(info auth.CreateAuthInfo) OutboundMiddlewareFunc {
	authClient := auth.Load(info)
	serviceName := info.Config().Get(config.ServiceNameKey).AsString()
	return func(req *http.Request, next Executor) (resp *http.Response, err error) {
		ctx := req.Context()
		// Client needs to know what service it is to authenticate
		authCtx := authClient.SetAttribute(ctx, auth.ServiceAuth, serviceName)

		authCtx, err = authClient.Authenticate(authCtx)
		if err != nil {
			ulog.Logger(ctx).Error(auth.ErrAuthentication, "error", err)
			return nil, err
		}

		span := opentracing.SpanFromContext(authCtx)
		if err := injectSpanIntoHeaders(req.Header, span); err != nil {
			ulog.Logger(authCtx).Error("Error injecting auth context", "error", err)
			return nil, err
		}

		return next.Execute(req.WithContext(authCtx))
	}
}

func injectSpanIntoHeaders(header http.Header, span opentracing.Span) error {
	carrier := opentracing.HTTPHeadersCarrier(header)
	if err := span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, carrier); err != nil {
		span.SetTag("error", err.Error())
		return err
	}
	return nil
}
