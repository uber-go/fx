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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/internal/fxcontext"
	"go.uber.org/fx/service"
	"go.uber.org/fx/uauth"

	"github.com/stretchr/testify/assert"
)

func TestFilterChain(t *testing.T) {
	host := service.NullHost()
	chain := newFilterChain([]Filter{}, getNoopHandler(host))
	response := testServeHTTP(chain)
	assert.True(t, strings.Contains(response.Body.String(), "filters ok"))
}

func TestFilterChainFilters(t *testing.T) {
	host := service.NullHost()
	chain := newFilterChain([]Filter{
		contextFilter(host),
		tracingServerFilter(host),
		authorizationFilter(host),
		panicFilter(host)},
		getNoopHandler(host))
	response := testServeHTTP(chain)
	assert.Contains(t, response.Body.String(), "filters ok")
}

func TestFilterChainFilters_AuthFailure(t *testing.T) {
	host := service.NullHost()
	uauth.UnregisterClient()
	uauth.RegisterClient(uauth.FakeFailureClient)
	uauth.SetupClient(host)
	defer uauth.UnregisterClient()
	defer uauth.SetupClient(host)
	chain := newFilterChain([]Filter{
		contextFilter(host),
		tracingServerFilter(host),
		authorizationFilter(host),
		panicFilter(host)},
		getNoopHandler(host))
	response := testServeHTTP(chain)
	assert.Contains(t, "Unauthorized access: Error authorizing the service", response.Body.String())
	assert.Equal(t, 401, response.Code)
}

func testServeHTTP(chain filterChain) *httptest.ResponseRecorder {
	request := httptest.NewRequest("", "http://filters", nil)
	response := httptest.NewRecorder()
	ctx := fxcontext.New(context.Background(), service.NullHost())
	chain.ServeHTTP(ctx, response, request)
	return response
}

func getNoopHandler(host service.Host) HandlerFunc {
	return func(ctx fx.Context, w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "filters ok")
	}
}
