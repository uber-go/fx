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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_testYaml = []byte(`
name: test
`)
	_testClient = New(fakeAuthInfo{yaml: _testYaml})
)

func TestNew(t *testing.T) {
	t.Parallel()
	chain, ok := _testClient.Transport.(executionChain)
	require.True(t, ok)
	assert.Equal(t, 2, len(chain.filters))
}

func TestNew_Panic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		New(fakeAuthInfo{yaml: []byte(``)})
	})
}

func TestClientDo(t *testing.T) {
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := _testClient.Do(req)
	checkOKResponse(t, resp, err)
}

func TestClientDoWithoutFilters(t *testing.T) {
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := _testClient.Do(req)
	checkOKResponse(t, resp, err)
}

func TestClientGet(t *testing.T) {
	svr := startServer()
	resp, err := _testClient.Get(svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientGetTwiceExecutesAllFilters(t *testing.T) {
	svr := startServer()
	count := 0
	var f FilterFunc = func(r *http.Request, next Executor) (resp *http.Response, err error) {
		count++
		return next.Execute(r)
	}

	cl := New(fakeAuthInfo{yaml: _testYaml}, f)
	resp, err := cl.Get(svr.URL)
	checkOKResponse(t, resp, err)
	require.Equal(t, 1, count)
	resp, err = cl.Get(svr.URL)
	checkOKResponse(t, resp, err)
	require.Equal(t, 2, count)
}

func TestClientGetError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _testClient.Get("%")
	checkErrResponse(t, resp, err)
}

func TestClientHead(t *testing.T) {
	svr := startServer()
	resp, err := _testClient.Head(svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientHeadError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _testClient.Head("%")
	checkErrResponse(t, resp, err)
}

func TestClientPost(t *testing.T) {
	svr := startServer()
	resp, err := _testClient.Post(svr.URL, "", nil)
	checkOKResponse(t, resp, err)
}

func TestClientPostError(t *testing.T) {
	resp, err := _testClient.Post("%", "", nil)
	checkErrResponse(t, resp, err)
}

func TestClientPostForm(t *testing.T) {
	svr := startServer()
	var urlValues map[string][]string
	resp, err := _testClient.PostForm(svr.URL, urlValues)
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
