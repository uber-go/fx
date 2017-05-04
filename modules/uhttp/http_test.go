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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"
	"time"

	"go.uber.org/fx/config"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

var _httpConfig = []byte(`
modules:
  uhttp:
    port: 0
    debug: true
`)

// Custom default client since http's defaultClient does not set timeout
var _defaultHTTPClient = &http.Client{Timeout: 2 * time.Second}

func TestNew_OK(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/something", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("funny"))
		assert.NoError(t, r.Body.Close())
	}))

	withModule(t, mux, func(m *Module) {
		resp, err := _defaultHTTPClient.Get(fmt.Sprintf("%s/something", getURL(m)))
		require.NoError(t, err)
		res, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, "funny", string(res))
	})
}

func TestHTTPModule_Panic_OK(t *testing.T) {
	t.Parallel()

	withModule(t, registerPanic(), func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode, "Expected 500 with panic wrapper")
		})
	})
}

func TestHTTPModule_Tracer(t *testing.T) {
	t.Parallel()
	withModule(t, registerTracerCheckHandler(), func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with tracer check")
		})
	})
}

func TestHTTPModule_StartsAndStops(t *testing.T) {
	t.Parallel()

	withModule(t, registerPanic(), func(m *Module) {
		assert.NotNil(t, m.listener, "Start should be successful")
	})
}

func TestBuiltinHealth_OK(t *testing.T) {
	t.Parallel()

	withModule(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/health", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with default health handler")
		})
	})
}

func TestPanicsOnNilConstructorHandler(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { New(nil) })
}

func TestNegativePortPanic(t *testing.T) {
	t.Parallel()

	di := dig.New()
	p := config.NewStaticProvider(map[string]interface{}{
		"modules": map[string]interface{}{
			"uhttp": map[string]interface{}{
				"port": -1,
			},
		},
	})

	di.MustRegister(&p)
	di.MustRegister(&tally.NoopScope)
	di.MustRegister(zap.NewNop())
	handlerCtor := func() (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), nil
	}

	mod := New(handlerCtor)
	ctors := mod.Constructor()
	for i := range ctors {
		di.MustRegister(ctors[i])
	}

	var s *starter
	err := di.Resolve(&s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to open TCP listener for HTTP module")

	runtime.Gosched()
	assert.NoError(t, mod.Stop(), "No exit error should occur")
}

func withModule(
	t testing.TB,
	handler http.Handler,
	fn func(*Module),
) {
	di := dig.New()
	p := config.NewYAMLProviderFromBytes(_httpConfig)
	di.MustRegister(&p)
	di.MustRegister(&tally.NoopScope)
	di.MustRegister(zap.NewNop())

	mod := New(&handler)
	ctors := mod.Constructor()
	for i := range ctors {
		di.MustRegister(ctors[i])
	}

	var s *starter
	di.MustResolve(&s)
	fn(mod)
	runtime.Gosched()
	assert.NoError(t, mod.Stop(), "No exit error should occur")
}

func getURL(m *Module) string {
	addr := m.listener.Addr()
	return fmt.Sprintf("http://%s", addr.String())
}

func makeRequest(m *Module, method, url string, body io.Reader, fn func(r *http.Response)) {
	base := getURL(m)
	request, err := http.NewRequest(method, base+url, body)
	if err != nil {
		// Yes, panics are OK for programmer errors in test suites
		panic(err)
	}

	response, err := _defaultHTTPClient.Do(request)
	if err != nil {
		panic(err)
	}
	fn(response)
}

func registerTracerCheckHandler() http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		if span == nil {
			panic(fmt.Sprintf("Intentional panic, invalid span: %v", span))
		} else if span.Tracer() != opentracing.GlobalTracer() {
			panic(fmt.Sprintf(
				"Intentional panic, expected tracer: %v different from actual tracer: %v", span.Tracer(),
				opentracing.GlobalTracer(),
			))
		}
	}
}

func registerPanic() http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		panic("Intentional panic for:" + r.URL.Path)
	}
}
