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

	"go.uber.org/fx"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/internal/fxcontext"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Executor executes the http request. Execute must be safe to use by multiple go routines
type Executor interface {
	Execute(ctx fx.Context, r *http.Request) (resp *http.Response, err error)
}

// Filter applies filters on client requests and request's context such as
// adding tracing to the context. Filters must call next.Execute() at most once, calling it twice and more
// will lead to an undefined behavior
type Filter interface {
	Apply(ctx fx.Context, r *http.Request, next Executor) (resp *http.Response, err error)
}

// FilterFunc is an adaptor to call normal functions to apply filters
type FilterFunc func(
	ctx fx.Context, r *http.Request, next Executor,
) (resp *http.Response, err error)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f FilterFunc) Apply(
	ctx fx.Context, r *http.Request, next Executor,
) (resp *http.Response, err error) {
	return f(ctx, r, next)
}

func tracingFilter() FilterFunc {
	return func(ctx fx.Context, req *http.Request, next Executor,
	) (resp *http.Response, err error) {
		opName := req.Method
		var parent opentracing.SpanContext
		if s := opentracing.SpanFromContext(ctx); s != nil {
			parent = s.Context()
		}
		span := opentracing.GlobalTracer().StartSpan(opName, opentracing.ChildOf(parent))
		ext.SpanKindRPCClient.Set(span)
		ext.HTTPUrl.Set(span, req.URL.String())
		defer span.Finish()

		ctx = &fxcontext.Context{
			Context: opentracing.ContextWithSpan(ctx, span),
		}
		if err := injectSpanIntoHeaders(req.Header, span); err != nil {
			return nil, err
		}

		resp, err = next.Execute(ctx, req)
		if resp != nil {
			span.SetTag("http.status_code", resp.StatusCode)
		}
		if err != nil {
			span.SetTag("error", err.Error())
		}
		return resp, err
	}
}

// authenticationFilter on client side calls authenticate, and gets a claim that client is who they say they are
// We only authorize with the claim on server side
func authenticationFilter(info auth.CreateAuthInfo) FilterFunc {
	authClient := auth.Load(info)
	serviceName := info.Config().Get(config.ServiceNameKey).AsString()
	return func(ctx fx.Context, req *http.Request, next Executor,
	) (resp *http.Response, err error) {
		// Client needs to know what service it is to authenticate
		authCtx := authClient.SetAttribute(ctx, auth.ServiceAuth, serviceName)

		authCtx, err = authClient.Authenticate(authCtx)
		if err != nil {
			ctx.Logger().Error(auth.ErrAuthentication, "error", err)
			return nil, err
		}

		span := opentracing.SpanFromContext(authCtx)
		if err := injectSpanIntoHeaders(req.Header, span); err != nil {
			ctx.Logger().Error("Error injecting auth context", "error", err)
			return nil, err
		}

		return next.Execute(&fxcontext.Context{Context: authCtx}, req)
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
