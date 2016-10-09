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
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/config"
	"github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/uberfx/modules"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// ModuleType is a human-friendly representation of the module
const ModuleType = "HTTP"

const (
	// ContentType is the header key that contains the body type
	ContentType = "Content-Type"
	// ContentLength is the length of the HTTP body
	ContentLength = "Content-Length"
	// ContentTypeText is the plain content type
	ContentTypeText = "text/plain"
	// ContentTypeJSON is the JSON content type
	ContentTypeJSON = "application/json"

	// HTTP defaults
	defaultTimeout = 60 * time.Second
	defaultPort    = 3001

	// Reporter timeout for tracking HTTP requests
	defaultReportTimeout = 90 * time.Second
)

var _ core.Module = &Module{}

// Response is an envelope for returning the results of an HTTP call
type Response struct {
	Status      int
	ContentType string
	Body        interface{}
	Headers     map[string]string
	Error       error
}

// A Module is a module to handle HTTP requests
type Module struct {
	modules.ModuleBase
	title    string
	config   Config
	mux      *http.ServeMux
	router   *mux.Router
	listener net.Listener
	handlers []RouteHandler
	listenMu sync.RWMutex
}

var _ core.Module = &Module{}

// Config handles config for HTTP modules
type Config struct {
	modules.ModuleConfig
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// CreateHTTPRegistrantsFunc returns a slice of registrants from a service host
type CreateHTTPRegistrantsFunc func(service core.ServiceHost) []RouteHandler

// NewHTTPModule returns a new HTTP module
func NewHTTPModule(hookup CreateHTTPRegistrantsFunc, options ...modules.Option) core.ModuleCreateFunc {
	return func(mi core.ModuleCreateInfo) ([]core.Module, error) {
		mod, err := newModule(mi, hookup, options...)
		if err != nil {
			return nil, err
		}
		return []core.Module{mod}, nil
	}
}
func newModule(mi core.ModuleCreateInfo, createService CreateHTTPRegistrantsFunc, options ...modules.Option) (*Module, error) {
	// setup config defaults
	cfg := &Config{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}

	for _, option := range options {
		option(&mi)
	}

	if mi.Name == "" {
		mi.Name = "http"
	}
	reporter := &metrics.LoggingTrafficReporter{Prefix: mi.Host.Name()}

	module := &Module{
		ModuleBase: *modules.NewModuleBase(ModuleType, mi.Name, mi.Host, reporter, []string{}),
		handlers:   createService(mi.Host),
	}

	config.Global().GetValue(getConfigKey(mi.Name)).PopulateStruct(cfg)
	module.config = *cfg
	return module, nil
}

// Initialize sets up an HTTP-backed module
func (m *Module) Initialize(host core.ServiceHost) error {
	return nil
}

// Start begins serving requests over HTTP
func (m *Module) Start(ready chan<- struct{}) <-chan error {
	m.mux = http.NewServeMux()
	// Do something unrelated to annotations
	m.router = mux.NewRouter()

	m.mux.Handle("/", m.router)

	for _, h := range m.handlers {
		handle := h.Handler
		handle = trackWrap(m.Reporter(), handle)
		handle = panicWrap(handle)
		// TODO other middlewares, logging, tracing?
		route := m.router.Handle(h.Path, handle)
		// apply all route options
		for _, opt := range h.Options {
			opt(route)
		}
	}

	ret := make(chan error, 1)

	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		ret <- err
		return ret
	}

	// finally, start the http server.
	// TODO use With/WithFields
	log.Printf("Server listening on port %d\n", m.config.Port)

	if err != nil {
		ret <- err
		return ret
	}
	m.listener = listener

	go func() {
		ret <- http.Serve(m.listener, m.mux)
	}()
	ready <- struct{}{}
	return ret
}

// Stop shuts down an HTTP module
func (m *Module) Stop() error {
	m.listenMu.Lock()
	defer m.listenMu.Unlock()

	if m.listener != nil {
		m.listener.Close()
		m.listener = nil
	}
	return nil
}

// IsRunning returns whether the module is currently running
func (m *Module) IsRunning() bool {
	m.listenMu.RLock()
	defer m.listenMu.RUnlock()

	return m.listener != nil
}

func getConfigKey(name string) string {
	return fmt.Sprintf("modules.%s", name)
}

// Middlewares

// handle any panics and return an error
func panicWrap(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// TODO log this
				w.Header().Add(ContentType, ContentTypeText)
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, fmt.Sprintf("Server error: %+v", err))
			}
		}()

		h.ServeHTTP(w, r)
	}
}

// track metrics per-request
func trackWrap(reporter metrics.TrafficReporter, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s_%s", r.Method, r.URL.Path)
		// TODO(ai) use sync pool to avoid allocations on every request
		data := map[string]string{
			metrics.TrafficCorrelationID: r.Header.Get("RequestID"),
		}
		tracker := reporter.Start(key, data, defaultReportTimeout)
		defer context.Clear(r)

		defer tracker.Finish("", "TODO", nil)
		// TODO(ai) get message and error from below
		h.ServeHTTP(w, r)
	})
}
