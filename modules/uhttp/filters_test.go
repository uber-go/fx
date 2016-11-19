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
	gcontext "context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
)

func TestExecutionChain(t *testing.T) {
	chain := newExecutionChain([]Filter{}, getNoopHandler())
	response := testServeHTTP(chain)
	assert.True(t, strings.Contains(response.Body.String(), "filters ok"))
}

func TestExecutionChainFilters(t *testing.T) {
	chain := newExecutionChain(
		[]Filter{tracerFilter{}, FilterFunc(panicFilter)},
		getNoopHandler(),
	)
	response := testServeHTTP(chain)
	assert.Contains(t, response.Body.String(), "filters ok")
}

func testServeHTTP(chain executionChain) *httptest.ResponseRecorder {
	request := httptest.NewRequest("", "http://filters", nil)
	response := httptest.NewRecorder()
	ctx := service.NewContext(gcontext.Background(), service.NullHost())
	chain.ServeHTTP(ctx, response, request)
	return response
}

func getNoopHandler() Handler {
	return HandlerFunc(func(ctx service.Context, w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "filters ok")
	})
}
