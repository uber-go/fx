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

type Option struct {
	graph dig.Graph
}

const _middlewareKey = "_httpClientMiddleware"
const _graphKey = "_httpClientGraph"

func WithOutbound(middleware... OutboundMiddleware) modules.Option{
	return func(info *service.ModuleCreateInfo) error {
		info.Items[_middlewareKey] = append(info.Items[_middlewareKey].([]OutboundMiddleware, middleware...)
		return nil
	}
}

func WithGraph(graph dig.Graph) modules.Option {
	return func(info *service.ModuleCreateInfo) error {
		info.Items[_graphKey] = graph
		return nil
	}
}

func New(options... modules.Option) *http.Client {
	var info service.ModuleCreateInfo
	for _, opt := range options {
		opt(info)
	}

	graph := dig.DefaultGraph()
	if g, ok := info.Items[_graphKey]; ok {
		graph = g.(dig.Graph)
	}

	middleware := make([]OutboundMiddleware, 0, len(options) + 2)

	var tracer opentracing.Tracer
	if err := graph.Resolve(&tracer); err != nil {

	}

	middleware = append(middleware, tracingOutbound(*tracer), authenticationOutbound(info))
	for _, x := range options {
		middleware = append(middleware, x())
	}

	return &http.Client{
		Transport: newExecutionChain(middleware, http.DefaultTransport),
		Timeout:   2 * time.Minute,
	}
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
