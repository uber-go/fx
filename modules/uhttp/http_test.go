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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"
	"time"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/modules/uhttp/internal/stats"
	"go.uber.org/fx/service"
	. "go.uber.org/fx/service/testutils"
	. "go.uber.org/fx/testutils"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

// Custom default client since http's defaultClient does not set timeout
var _defaultHTTPClient = &http.Client{Timeout: 2 * time.Second}

func TestNew_OK(t *testing.T) {
	WithService(New(registerNothing), nil, []service.Option{configOption()}, func(s service.Owner) {
		assert.NotNil(t, s, "Should create a module")
	})
}

func TestNew_WithOptions(t *testing.T) {
	options := []modules.Option{
		modules.WithRoles("testing"),
	}

	withModule(t, registerPanic, options, false, func(m *Module) {
		assert.NotNil(t, m, "Expected OK with options")
	})
}

func TestHTTPModule_WithFilters(t *testing.T) {
	withModule(t, registerPanic, []modules.Option{WithFilters(fakeFilter())}, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Contains(t, string(body), "filter is executed")
		})
		verifyMetrics(t, m.Host().Metrics())
	})
}

func TestHTTPModule_WithUserPanicFilter(t *testing.T) {
	withModule(t, registerTracerCheckHandler, []modules.Option{WithFilters(userPanicFilter())}, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode, "Expected 500 with panic wrapper")
		})
	})
}

func TestHTTPModule_Panic_OK(t *testing.T) {
	withModule(t, registerPanic, nil, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode, "Expected 500 with panic wrapper")
		})
	})
}

func TestHTTPModule_Tracer(t *testing.T) {
	withModule(t, registerTracerCheckHandler, nil, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with tracer check")
		})
	})
}

func TestHTTPModule_StartsAndStops(t *testing.T) {
	withModule(t, registerPanic, nil, false, func(m *Module) {
		assert.True(t, m.IsRunning(), "Start should be successful")
	})
}

func TestBuiltinHealth_OK(t *testing.T) {
	withModule(t, registerNothing, nil, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/health", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with default health handler")
		})
	})
}

func TestOverrideHealth_OK(t *testing.T) {
	withModule(t, registerCustomHealth, nil, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/health", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with default health handler")
			body, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err, "Should be able to read health body")
			assert.Equal(t, "not ok", string(body))
		})
	})
}

func TestPProf_Registered(t *testing.T) {
	withModule(t, registerNothing, nil, false, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/debug/pprof", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 from pprof handler")
		})
	})
}

func TestHookupOptions(t *testing.T) {
	options := []modules.Option{
		modules.WithName("an optional name"),
	}

	withModule(t, registerNothing, options, false, func(m *Module) {
		assert.NotNil(t, m)
	})
}

func TestHookupOptions_Error(t *testing.T) {
	options := []modules.Option{
		func(_ *service.ModuleCreateInfo) error {
			return errors.New("i just can't")
		},
	}

	withModule(t, registerNothing, options, true, func(m *Module) {
		assert.Nil(t, m)
	})
}

// TODO(ai) add a test for binding a bad port and get an error out of Start()

func configOption() service.Option {
	return service.WithConfiguration(StaticAppData(nil))
}

func withModule(
	t testing.TB,
	hookup GetHandlersFunc,
	options []modules.Option,
	expectError bool,
	fn func(*Module),
) {
	mi := service.ModuleCreateInfo{
		Host: service.NopHost(),
	}
	stats.SetupHTTPMetrics(mi.Host.Metrics())
	mod, err := newModule(mi, hookup, options...)
	if expectError {
		require.Error(t, err, "Expected error instantiating module")
		fn(nil)
		return
	}

	require.NoError(t, err, "Unable to instantiate module")

	// us an ephemeral port on tests
	mod.config.Port = 0

	errs := make(chan error, 1)
	readyChan := make(chan struct{}, 1)
	go func() {
		// horray funny channel syntax. start() returns an err chan
		errs <- <-mod.Start(readyChan)
	}()
	select {
	case <-readyChan:
	// cool, we're ready
	case <-time.After(time.Second):
		assert.Fail(t, "Module failed to start after 1 second")
	}

	var exitError error
	defer func() {
		exitError = mod.Stop()
	}()

	fn(mod)
	runtime.Gosched()
	assert.NoError(t, exitError, "No exit error should occur")
	// check errs channel
	select {
	case err := <-errs:
		assert.NoError(t, err, "Got error from listening")
	default:
		// no errors, we're good
	}
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

func registerNothing(_ service.Host) []RouteHandler {
	return nil
}

func makeSingleHandler(path string, fn func(http.ResponseWriter, *http.Request)) []RouteHandler {
	return []RouteHandler{
		{
			Path:    path,
			Handler: http.HandlerFunc(fn),
		},
	}
}

func registerTracerCheckHandler(host service.Host) []RouteHandler {
	return makeSingleHandler("/", func(_ http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		if span == nil {
			panic(fmt.Sprintf("Intentional panic, invalid span: %v", span))
		} else if span.Tracer() != opentracing.GlobalTracer() {
			panic(fmt.Sprintf(
				"Intentional panic, expected tracer: %v different from actual tracer: %v", span.Tracer(),
				opentracing.GlobalTracer(),
			))
		}
	})
}

func registerCustomHealth(_ service.Host) []RouteHandler {
	return makeSingleHandler("/health", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "not ok")
	})
}

func registerPanic(_ service.Host) []RouteHandler {
	return makeSingleHandler("/", func(_ http.ResponseWriter, r *http.Request) {
		panic("Intentional panic for:" + r.URL.Path)
	})
}

func fakeFilter() FilterFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		io.WriteString(w, "filter is executed")
		next.ServeHTTP(w, r)
	}
}

func userPanicFilter() FilterFunc {
	return func(_ http.ResponseWriter, r *http.Request, _ http.Handler) {
		panic("Intentional panic for:" + r.URL.Path)
	}
}

func verifyMetrics(t *testing.T, scope tally.Scope) {
	snapshot := scope.(tally.TestScope).Snapshot()
	timers := snapshot.Timers()
	counters := snapshot.Counters()

	assert.NotNil(t, timers["GET"].Values())
	assert.NotNil(t, counters["fail"].Value())
}
