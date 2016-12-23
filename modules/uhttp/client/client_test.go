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
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/stretchr/testify/assert"
)

var _defaultHTTPClient = &http.Client{Timeout: 2 * time.Second}
var _defaultUHTTPClient = New(_defaultHTTPClient)

func TestNew(t *testing.T) {
	uhttpClient := New(_defaultHTTPClient)
	assert.Equal(t, _defaultHTTPClient, uhttpClient.Client)
	assert.Equal(t, 2, len(uhttpClient.filters))
}

func TestClientDo(t *testing.T) {
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := _defaultUHTTPClient.Do(fx.NoopContext, req)
	checkOKResponse(t, resp, err)
}

func TestClientDoWithoutFilters(t *testing.T) {
	uhttpClient := &Client{Client: _defaultHTTPClient}
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := uhttpClient.Do(fx.NoopContext, req)
	checkOKResponse(t, resp, err)
}

func TestClientGet(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Get(fx.NoopContext, svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientGetError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _defaultUHTTPClient.Get(fx.NoopContext, "%")
	checkErrResponse(t, resp, err)
}

func TestClientHead(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Head(fx.NoopContext, svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientHeadError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _defaultUHTTPClient.Head(fx.NoopContext, "%")
	checkErrResponse(t, resp, err)
}

func TestClientPost(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Post(fx.NoopContext, svr.URL, "", nil)
	checkOKResponse(t, resp, err)
}

func TestClientPostError(t *testing.T) {
	resp, err := _defaultUHTTPClient.Post(fx.NoopContext, "%", "", nil)
	checkErrResponse(t, resp, err)
}

func TestClientPostForm(t *testing.T) {
	svr := startServer()
	var urlValues map[string][]string
	resp, err := _defaultUHTTPClient.PostForm(fx.NoopContext, svr.URL, urlValues)
	checkOKResponse(t, resp, err)
}

func checkErrResponse(t *testing.T, resp *http.Response, err error) {
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func checkOKResponse(t *testing.T, resp *http.Response, err error) {
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func startServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
}

func createHTTPClientRequest(url string) *http.Request {
	req := httptest.NewRequest("", url, nil)
	// To prevent http: Request.RequestURI can't be set in client requests
	req.RequestURI = ""
	return req
}
