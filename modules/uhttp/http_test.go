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
	"net/http"
	"runtime"
	"testing"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/service"
	. "go.uber.org/fx/service/testutils"
	. "go.uber.org/fx/testutils"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

var _httpconfig = []byte(`
modules:
  uhttp:
    port: 0
    debug: true
`)

// Custom default client since http's defaultClient does not set timeout
var _defaultHTTPClient = &http.Client{Timeout: 2 * time.Second}

func TestNew_OK(t *testing.T) {
	WithService(New(registerNothing()), nil, []service.Option{configOption()}, func(s service.Manager) {
		assert.NotNil(t, s, "Should create a module")

	})
}

func TestHTTPModule_Panic_OK(t *testing.T) {
	withModule(t, registerPanic(), false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode, "Expected 500 with panic wrapper")
		})
	})
}

func TestHTTPModule_Tracer(t *testing.T) {
	withModule(t, registerTracerCheckHandler(), false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with tracer check")
		})
	})
}

func TestHTTPModule_StartsAndStops(t *testing.T) {
	withModule(t, registerPanic(), false, func(m *Module) {
		assert.NotNil(t, m.listener, "Start should be successful")
	})
}

func TestBuiltinHealth_OK(t *testing.T) {
	withModule(t, registerNothing(), false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/health", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with default health handler")
		})
	})
}

// TODO(ai) add a test for binding a bad port and get an error out of Start()

func configOption() service.Option {
	return service.WithConfiguration(StaticAppData(nil))
}

func withModule(
	t testing.TB,
	handler http.Handler,
	expectError bool,
	fn func(*Module),
) {
	host, err := service.NewScopedHost(
		service.NopHostWithConfig(
			config.NewYAMLProviderFromBytes(_httpconfig)),
		"uhttp",
		"hello",
	)
	require.NoError(t, err)
	mod, err := newModule(host, handler)
	if expectError {
		require.Error(t, err, "Expected error instantiating module")
		fn(nil)
		return
	}
	require.NoError(t, err, "Unable to instantiate module")
	assert.NoError(t, mod.Start(), "Got error from starting")
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

func registerNothing() http.Handler {
	return nil
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

func verifyMetrics(t *testing.T, scope tally.Scope) {
	snapshot := scope.(tally.TestScope).Snapshot()
	timers := snapshot.Timers()
	counters := snapshot.Counters()

	require.NotNil(t, timers["GET"])
	assert.NotNil(t, timers["GET"].Values())
	require.NotNil(t, counters["fail"])
	assert.NotNil(t, counters["fail"].Value())
}
