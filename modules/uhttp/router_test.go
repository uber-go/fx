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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/service"
)

func serve(t *testing.T, h http.Handler) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)

	go http.Serve(l, h)
	return l
}

func withRouter(t *testing.T, f func(r *Router, l net.Listener)) {
	r := NewRouter(service.NullHost())
	l := serve(t, r)
	defer l.Close()
	r.Handle("/foo/baz/quokka",
		HandlerFunc(func(ctx service.Context, w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		}))
	r.Handle("/foo/bar/quokka",
		HandlerFunc(func(ctx service.Context, w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("world"))
		}))
	f(r, l)
}

func TestRouting_ExpectSecond(t *testing.T) {
	withRouter(t, func(r *Router, l net.Listener) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/foo/bar/quokka", l.Addr().String()), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "world", string(body))
	})
}

func TestRouting_ExpectFirst(t *testing.T) {
	withRouter(t, func(r *Router, l net.Listener) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/foo/baz/quokka", l.Addr().String()), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(body))
	})
}
