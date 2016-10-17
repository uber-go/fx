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

package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/core"
	. "go.uber.org/fx/core/testutils"
	"go.uber.org/fx/modules"
)

func TestNewHTTPModule_OK(t *testing.T) {
	defer WithConfig(nil)()
	WithService(NewHTTPModule(registerNothing), nil, func(s core.ServiceOwner) {
		assert.NotNil(t, s, "Should create a module")
	})
}

func TestNewHTTPModule_WithOptions(t *testing.T) {
	defer WithConfig(nil)()

	options := []modules.Option{
		modules.WithRoles("testing"),
	}

	withModule(t, registerPanic, options, func(m *Module) {
		assert.NotNil(t, m, "Expected OK with options")
	})
}

func TestHTTPModule_Panic_OK(t *testing.T) {
	withModule(t, registerPanic, nil, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode, "Expected 500 with panic  wrapper")
		})
	})
}

func TestHTTPModule_StartsAndStops(t *testing.T) {
	withModule(t, registerPanic, nil, func(m *Module) {
		assert.True(t, m.IsRunning(), "Start should be successful")
	})
}

func TestBuiltinHealth_OK(t *testing.T) {
	withModule(t, registerNothing, nil, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/health", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 with default health handler")
		})
	})
}

func TestOverrideHealth_OK(t *testing.T) {
	withModule(t, registerCustomHealth, nil, func(m *Module) {
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
	withModule(t, registerNothing, nil, func(m *Module) {
		assert.NotNil(t, m)
		makeRequest(m, "GET", "/debug/pprof", nil, func(r *http.Response) {
			assert.Equal(t, http.StatusOK, r.StatusCode, "Expected 200 from pprof handler")
		})
	})
}

// TODO(ai) add a test for binding a bad port and get an error out of Start()

func withModule(t testing.TB, hookup CreateHTTPRegistrantsFunc, options []modules.Option, fn func(*Module)) {
	defer WithConfig(nil)()
	mi := core.ModuleCreateInfo{
		Host: core.NullServiceHost(),
	}
	mod, err := newModule(mi, hookup, options...)
	require.NoError(t, err, "Unable to instantiate module")

	// us an ephemeral port on tests
	mod.config.Port = 0
	require.NoError(t, mod.Initialize(mi.Host), "Expected initialize to succeed")

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
	case <-errs:
		assert.Fail(t, "Got error from listening")
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

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	fn(response)
}

func registerNothing(_ core.ServiceHost) []RouteHandler {
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

func registerCustomHealth(_ core.ServiceHost) []RouteHandler {
	return makeSingleHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "not ok")
	})
}

func registerPanic(_ core.ServiceHost) []RouteHandler {
	return makeSingleHandler("/", func(_ http.ResponseWriter, r *http.Request) {
		panic("Intentional panic for:" + r.URL.Path)
	})
}
