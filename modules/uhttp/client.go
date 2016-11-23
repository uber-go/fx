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

package uhttp

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/fx/core"

	"golang.org/x/net/context/ctxhttp"
)

// Client wraps around a http client
type Client struct {
	http.Client
	filters []ClientFilter
}

// NewClient creates a new instance of uhttp Client
func NewClient(client http.Client, filters ...ClientFilter) *Client {
	filters = append(filters, ClientFilterFunc(tracingClientFilter))
	return &Client{Client: client, filters: filters}
}

// Do is a context-aware, filter-enabled extension of Do() in http.Client
func (c *Client) Do(ctx core.Context, req *http.Request) (resp *http.Response, err error) {
	filters := c.filters
	// TODO: Need a way to handle the case where Client is initialized without the NewClient method
	// and some filters are set. Need to always include the tracing filter
	if len(filters) == 0 {
		filters = append(filters, ClientFilterFunc(tracingClientFilter))
	}
	execChain := newClientExecutionChain(filters, BasicClientFunc(c.do))
	return execChain.Do(ctx, req)
}

func (c *Client) do(ctx core.Context, req *http.Request) (resp *http.Response, err error) {
	return ctxhttp.Do(ctx, &c.Client, req)
}

// Get is a context-aware, filter-enabled extension of Get() in http.Client
func (c *Client) Get(ctx core.Context, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post is a context-aware, filter-enabled extension of Post() in http.Client
func (c *Client) Post(
	ctx core.Context,
	url string,
	bodyType string,
	body io.Reader,
) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(ctx, req)
}

// PostForm is a context-aware, filter-enabled extension of PostForm() in http.Client
func (c *Client) PostForm(
	ctx core.Context,
	url string,
	data url.Values,
) (resp *http.Response, err error) {
	return c.Post(ctx, url, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
}

// Head is a context-aware, filter-enabled extension of Head() in http.Client
func (c *Client) Head(ctx core.Context, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// BasicClient is the simplest, context-aware HTTP client with a single method Do.
type BasicClient interface {
	// Do sends an HTTP request and returns an HTTP response, following
	// policy (e.g. redirects, cookies, auth) as configured on the client.
	Do(ctx core.Context, req *http.Request) (resp *http.Response, err error)
}

// The BasicClientFunc type is an adapter to allow the use of ordinary functions as BasicClient.
type BasicClientFunc func(ctx core.Context, req *http.Request) (resp *http.Response, err error)

// Do implements Do from the BasicClient interface
func (f BasicClientFunc) Do(
	ctx core.Context, req *http.Request,
) (resp *http.Response, err error) {
	return f(ctx, req)
}
