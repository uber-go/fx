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

	"go.uber.org/fx/auth"
	"go.uber.org/fx/dig"
	"go.uber.org/fx/modules"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_testYaml = []byte(`
name: test
`)
	_tracer   = opentracing.NoopTracer{}
	_authInfo = fakeAuthInfo{yaml: _testYaml}
)

func withTestGraph(t *testing.T, tracer opentracing.Tracer, info auth.CreateAuthInfo) modules.Option {
	g := dig.New()
	require.NoError(t, g.Register(&tracer))
	require.NoError(t, g.Register(&info))
	return WithGraph(g)
}

func testClient(t *testing.T) *http.Client {
	cl, err := New(withTestGraph(t, _tracer, _authInfo))
	require.NoError(t, err)
	return cl
}

func TestNew(t *testing.T) {
	t.Parallel()

	chain, ok := testClient(t).Transport.(executionChain)
	require.True(t, ok)
	assert.Equal(t, 2, len(chain.middleware))
}

func TestNew_Panic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		New(withTestGraph(t, _tracer, &fakeAuthInfo{yaml: []byte(``)}))
	})
}

func TestClientDo(t *testing.T) {
	t.Parallel()
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := testClient(t).Do(req)
	checkOKResponse(t, resp, err)
}

func TestClientDoWithoutMiddleware(t *testing.T) {
	t.Parallel()
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := testClient(t).Do(req)
	checkOKResponse(t, resp, err)
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	svr := startServer()
	resp, err := testClient(t).Get(svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientGetTwiceExecutesAllMiddleware(t *testing.T) {
	t.Parallel()
	svr := startServer()
	count := 0
	var f OutboundMiddlewareFunc = func(r *http.Request, next Executor) (resp *http.Response, err error) {
		count++
		return next.Execute(r)
	}

	cl, err := New(withTestGraph(t, _tracer, _authInfo), WithOutbound(f), WithOutbound(f))
	require.NoError(t, err)
	resp, err := cl.Get(svr.URL)
	checkOKResponse(t, resp, err)
	require.Equal(t, 2, count)
	resp, err = cl.Get(svr.URL)
	checkOKResponse(t, resp, err)
	require.Equal(t, 4, count)
}

func TestClientGetError(t *testing.T) {
	t.Parallel()
	// Causing newRequest to fail, % does not parse as URL
	resp, err := testClient(t).Get("%")
	checkErrResponse(t, resp, err)
}

func TestClientHead(t *testing.T) {
	t.Parallel()
	svr := startServer()
	resp, err := testClient(t).Head(svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientHeadError(t *testing.T) {
	t.Parallel()
	// Causing newRequest to fail, % does not parse as URL
	resp, err := testClient(t).Head("%")
	checkErrResponse(t, resp, err)
}

func TestClientPost(t *testing.T) {
	t.Parallel()
	svr := startServer()
	resp, err := testClient(t).Post(svr.URL, "", nil)
	checkOKResponse(t, resp, err)
}

func TestClientPostError(t *testing.T) {
	t.Parallel()
	resp, err := testClient(t).Post("%", "", nil)
	checkErrResponse(t, resp, err)
}

func TestClientPostForm(t *testing.T) {
	t.Parallel()
	svr := startServer()
	var urlValues map[string][]string
	resp, err := testClient(t).PostForm(svr.URL, urlValues)
	checkOKResponse(t, resp, err)
}

func TestClientConstructionErrors(t *testing.T) {
	t.Parallel()
	g := dig.New()
	_, err := New(WithGraph(g))
	var tracer *opentracing.Tracer
	require.Equal(t, err, g.Resolve(&tracer))
	require.NoError(t, g.Register(tracer))
	var info *auth.CreateAuthInfo
	_, err = New(WithGraph(g))
	require.Equal(t, err, g.Resolve(&info))
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
