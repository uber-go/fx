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
	errClient = errors.New("client test error")
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
