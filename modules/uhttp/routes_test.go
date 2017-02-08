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

package uhttp

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestFromGorilla_OK(t *testing.T) {
	r := mux.NewRouter()
	route := r.Headers("foo", "bar")
	f := FromGorilla(route)
	assert.Equal(t, f.r, route)
}

func TestNewRouteHandler(t *testing.T) {
	rh := NewRouteHandler("/", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi\n")
	}))

	assert.Equal(t, rh.Path, "/")
}

func TestGorillaMux_OK(t *testing.T) {
	r := mux.NewRouter()
	route := r.Path("/foo")
	ours := FromGorilla(route)
	rounded := ours.GorillaMux()
	assert.Equal(t, route, rounded)
}

func TestHeaders_OK(t *testing.T) {
	r := mux.NewRouter()
	route := Route{r.Path("/foo")}
	withHeaders := route.Headers("foo", "bar")
	assert.NotNil(t, withHeaders.r)
}

func TestMethods_OK(t *testing.T) {
	r := mux.NewRouter()
	route := Route{r.Path("/foo")}
	withMethods := route.Methods("GET")
	assert.NotNil(t, withMethods.r)
}
