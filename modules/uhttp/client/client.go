// Copyright (c) 2016 Uber Technologies, Inc.
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
	"go.uber.org/fx/internal/fxcontext"
)

// New creates an http.Client that includes 2 extra filters: tracing and auth
// they are going to be applied in following order: tracing, auth, remaining filters
// and only if all of them passed the request is going to be send.
func New(info auth.CreateAuthInfo, filters ...Filter) *http.Client {
	extra := make([]Filter, len(filters)+2)
	extra[0] = tracingFilter()
	extra[1] = authenticationFilter(info)
	copy(extra[2:], filters)

	return &http.Client{
		Transport: newExecutionChain(extra, http.DefaultTransport),
	}
}

func newExecutionChain(
	filters []Filter, finalTransport http.RoundTripper,
) executionChain {
	return executionChain{
		filters:        filters,
		finalTransport: finalTransport,
	}
}

type executionChain struct {
	currentFilter  int
	filters        []Filter
	finalTransport http.RoundTripper
}

func (ec executionChain) Send(ctx fx.Context, r *http.Request) (resp *http.Response, err error) {
	if ec.currentFilter < len(ec.filters) {
		filter := ec.filters[ec.currentFilter]
		ec.currentFilter++

		return filter.Apply(ctx, r, ec)
	}

	return ec.finalTransport.RoundTrip(r.WithContext(ctx))
}

func (ec executionChain) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	return ec.Send(&fxcontext.Context{Context: r.Context()}, r)
}
