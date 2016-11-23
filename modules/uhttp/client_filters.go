package uhttp

import (
	"net/http"

	"go.uber.org/fx/core"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// ClientFilter applies filters on client requests, request contexts such as
// adding tracing to the context
type ClientFilter interface {
	Apply(ctx core.Context, r *http.Request, next BasicClient) (resp *http.Response, err error)
}

// ClientFilterFunc is an adaptor to call normal functions to apply filters
type ClientFilterFunc func(
	ctx core.Context, r *http.Request, next BasicClient,
) (resp *http.Response, err error)

// Apply implements Apply from the Filter interface and simply delegates to the function
func (f ClientFilterFunc) Apply(
	ctx core.Context, r *http.Request, next BasicClient,
) (resp *http.Response, err error) {
	return f(ctx, r, next)
}

func tracingClientFilter(
	ctx core.Context, req *http.Request, next BasicClient,
) (resp *http.Response, err error) {
	opName := req.Method
	var parent opentracing.SpanContext
	if s := opentracing.SpanFromContext(ctx); s != nil {
		parent = s.Context()
	}
	span := ctx.Tracer().StartSpan(opName, opentracing.ChildOf(parent))
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, req.URL.String())
	defer span.Finish()

	ctx = ctx.WithContext(opentracing.ContextWithSpan(ctx, span))
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, carrier)

	resp, err = next.Do(ctx, req)
	if resp != nil {
		span.SetTag("http.status_code", resp.StatusCode)
	}
	if err != nil {
		span.SetTag("error", err.Error())
	}
	return resp, err
}

func newClientExecutionChain(
	filters []ClientFilter, finalClient BasicClient,
) clientExecutionChain {
	return clientExecutionChain{
		filters:     filters,
		finalClient: finalClient,
	}
}

type clientExecutionChain struct {
	currentFilter int
	filters       []ClientFilter
	finalClient   BasicClient
}

func (ec clientExecutionChain) Do(
	ctx core.Context, req *http.Request,
) (resp *http.Response, err error) {
	if ec.currentFilter < len(ec.filters) {
		filter := ec.filters[ec.currentFilter]
		ec.currentFilter++
		return filter.Apply(ctx, req, ec)
	}
	return ec.finalClient.Do(ctx, req)
}
