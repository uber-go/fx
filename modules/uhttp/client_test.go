package uhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/core"
	"go.uber.org/fx/service"
	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
)

var _defaultUHTTPClient = NewClient(*_defaultHTTPClient)

func TestNewClient(t *testing.T) {
	uhttpClient := NewClient(*_defaultHTTPClient)
	assert.Equal(t, *_defaultHTTPClient, uhttpClient.Client)
	assert.Equal(t, 1, len(uhttpClient.filters))
}

func TestClientDo(t *testing.T) {
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := _defaultUHTTPClient.Do(createContext(), req)
	checkOKResponse(t, resp, err)
}

func TestClientDoWithoutFilters(t *testing.T) {
	uhttpClient := &Client{Client: *_defaultHTTPClient}
	svr := startServer()
	req := createHTTPClientRequest(svr.URL)
	resp, err := uhttpClient.Do(createContext(), req)
	checkOKResponse(t, resp, err)
}

func TestClientGet(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Get(createContext(), svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientGetError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _defaultUHTTPClient.Get(createContext(), "%")
	checkErrResponse(t, resp, err)
}

func TestClientHead(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Head(createContext(), svr.URL)
	checkOKResponse(t, resp, err)
}

func TestClientHeadError(t *testing.T) {
	// Causing newRequest to fail, % does not parse as URL
	resp, err := _defaultUHTTPClient.Head(createContext(), "%")
	checkErrResponse(t, resp, err)
}

func TestClientPost(t *testing.T) {
	svr := startServer()
	resp, err := _defaultUHTTPClient.Post(createContext(), svr.URL, "", nil)
	checkOKResponse(t, resp, err)
}

func TestClientPostError(t *testing.T) {
	resp, err := _defaultUHTTPClient.Post(createContext(), "%", "", nil)
	checkErrResponse(t, resp, err)
}

func TestClientPostForm(t *testing.T) {
	svr := startServer()
	var urlValues map[string][]string
	resp, err := _defaultUHTTPClient.PostForm(createContext(), svr.URL, urlValues)
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

func createContext() core.Context {
	return core.NewContext(context.Background(), service.NullHost())
}
