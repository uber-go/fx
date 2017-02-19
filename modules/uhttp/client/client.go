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
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/dig"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

const (
	_middlewareKey = "httpClientMiddleware"
	_graphKey      = "httpClientGraph"
)

// WithOutbound lets you add custom middleware to the client
func WithOutbound(middleware ...OutboundMiddleware) modules.Option {
	return func(info *service.ModuleCreateInfo) error {
		items := info.Items
		if m, ok := items[_middlewareKey]; ok {
			items[_middlewareKey] = append(m.([]OutboundMiddleware), middleware...)
		} else {
			items[_middlewareKey] = middleware
		}
		return nil
	}
}

// WithGraph allows you to a custom dependency injection graph to resolve Tracer and AuthInfo
func WithGraph(graph dig.Graph) modules.Option {
	return func(info *service.ModuleCreateInfo) error {
		info.Items[_graphKey] = graph
		return nil
	}
}

// New creates an http.Client that includes 2 extra outbound middleware: tracing and auth
// they are going to be applied in following order: tracing, auth, remaining outbound middleware
// and only if all of them passed the request is going to be send.
func New(options ...modules.Option) (*http.Client, error) {
	info := service.ModuleCreateInfo{
		Items: make(map[string]interface{}),
	}

	for _, opt := range options {
		opt(&info)
	}

	graph := dig.DefaultGraph()
	if g, ok := info.Items[_graphKey]; ok {
		graph = g.(dig.Graph)
	}

	var tracer *opentracing.Tracer
	if err := graph.Resolve(&tracer); err != nil {
		return nil, err
	}

	var auth *auth.CreateAuthInfo
	if err := graph.Resolve(&auth); err != nil {
		return nil, err
	}

	middleware := make([]OutboundMiddleware, 0, len(options)+2)
	middleware = append(middleware, tracingOutbound(*tracer), authenticationOutbound(*auth))

	if val, ok := info.Items[_middlewareKey]; ok {
		middleware = append(middleware, val.([]OutboundMiddleware)...)
	}

	return &http.Client{
		Transport: newExecutionChain(middleware, http.DefaultTransport),
		Timeout:   2 * time.Minute,
	}, nil
}

// executionChain represents a chain of outbound middleware that are being executed recursively
// in the increasing order middleware[0], middleware[1], ... The final transport is called
// to make RoundTrip after the last middleware is completed.
type executionChain struct {
	currentMiddleware int
	middleware        []OutboundMiddleware
	finalTransport    http.RoundTripper
}

func newExecutionChain(
	middleware []OutboundMiddleware, finalTransport http.RoundTripper,
) executionChain {
	return executionChain{
		middleware:     middleware,
		finalTransport: finalTransport,
	}
}

func (ec executionChain) Execute(r *http.Request) (resp *http.Response, err error) {
	if ec.currentMiddleware < len(ec.middleware) {
		middleware := ec.middleware[ec.currentMiddleware]
		ec.currentMiddleware++

		return middleware.Handle(r, ec)
	}

	return ec.finalTransport.RoundTrip(r)
}

// Implement http.RoundTripper interface to use as a Transport in http.Client
func (ec executionChain) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	return ec.Execute(r)
}
