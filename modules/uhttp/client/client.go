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
	"sync"

	"go.uber.org/fx"
)

var (
	_uFilters      sync.Mutex
	_filterTripper filterTripper
)

type filterTripper struct {
	oldTripper http.RoundTripper
	chain      executionChain
}

// UpdateDefaultFilters updates filters for the default RoundTripper
func UpdateDefaultFilters(filters ...Filter) {
	_uFilters.Lock()
	_filterTripper.chain = newExecutionChain(filters, _filterTripper.oldTripper)
	_uFilters.Unlock()
}

func (t *filterTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	return t.oldTripper.RoundTrip(req)
}

func init() {
	http.DefaultTransport = &filterTripper{oldTripper: http.DefaultTransport}
}

// The BasicClientFunc type is an adapter to allow the use of ordinary functions as BasicClient.
type BasicClientFunc func(ctx fx.Context, req *http.Request) (resp *http.Response, err error)

// RoundTrip implements RoundTrip from the http.RoundTripper interface
func (f BasicClientFunc) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return f(req.Context().(fx.Context), req)
}
