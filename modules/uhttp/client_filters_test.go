package uhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/core"

	"github.com/stretchr/testify/assert"
)

var (
	_respOK   = &http.Response{StatusCode: http.StatusOK}
	_req      = httptest.NewRequest("", "http://localhost", nil)
	errClient = errors.New("Client error")
)

func TestClientExecutionChain(t *testing.T) {
	execChain := newClientExecutionChain([]ClientFilter{}, getNoopClient())
	resp, err := execChain.Do(nil, _req)
	assert.NoError(t, err)
	assert.Equal(t, _respOK, resp)
}

func TestClientExecutionChainFilters(t *testing.T) {
	execChain := newClientExecutionChain(
		[]ClientFilter{ClientFilterFunc(tracingClientFilter)}, getNoopClient(),
	)
	ctx := createContext()
	resp, err := execChain.Do(ctx, _req)
	assert.NoError(t, err)
	assert.Equal(t, _respOK, resp)
}

func TestClientExecutionChainFiltersError(t *testing.T) {
	execChain := newClientExecutionChain(
		[]ClientFilter{ClientFilterFunc(tracingClientFilter)}, getErrorClient(),
	)
	resp, err := execChain.Do(createContext(), _req)
	assert.Error(t, err)
	assert.Equal(t, errClient, err)
	assert.Nil(t, resp)
}

func getNoopClient() BasicClient {
	return BasicClientFunc(
		func(ctx core.Context, req *http.Request) (resp *http.Response, err error) {
			return _respOK, nil
		},
	)
}

func getErrorClient() BasicClient {
	return BasicClientFunc(
		func(ctx core.Context, req *http.Request) (resp *http.Response, err error) {
			return nil, errClient
		},
	)
}
