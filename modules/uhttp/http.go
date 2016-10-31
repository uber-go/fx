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
	"net"
	"net/http"
	_ "net/http/pprof" // for automatic pprof
	"sync"
	"time"

	"go.uber.org/fx/core/ulog"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

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

	// default healthcheck endpoint
	healthPath = "/health"
)

var _ service.Module = &Module{}

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
	log      ulog.Log
	mux      *http.ServeMux
	router   *mux.Router
	listener net.Listener
	handlers []RouteHandler
	listenMu sync.RWMutex
}

var _ service.Module = &Module{}

// Config handles config for HTTP modules
type Config struct {
	modules.ModuleConfig
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
	Debug   *bool         `yaml:"debug"`
}

// CreateHTTPRegistrantsFunc returns a slice of registrants from a service host
type CreateHTTPRegistrantsFunc func(service service.Host) []RouteHandler

// New returns a new HTTP module
func New(hookup CreateHTTPRegistrantsFunc, options ...modules.Option) service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod, err := newModule(mi, hookup, options...)
		if err != nil {
			return nil, err
		}
		return []service.Module{mod}, nil
	}
}

func newModule(
	mi service.ModuleCreateInfo,
	createService CreateHTTPRegistrantsFunc,
	options ...modules.Option,
) (*Module, error) {
	// setup config defaults
	cfg := &Config{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}

	if mi.Name == "" {
		mi.Name = "http"
	}

	module := &Module{
		ModuleBase: *modules.NewModuleBase(ModuleType, mi.Name, mi.Host, []string{}),
		handlers:   createService(mi.Host),
	}

	module.Host().Config().GetValue(getConfigKey(mi.Name)).PopulateStruct(cfg)
	module.config = *cfg

	module.log = ulog.Logger().With("moduleName", mi.Name)

	for _, option := range options {
		if err := option(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", option)
			return module, err
		}
	}

	return module, nil
}

// Initialize sets up an HTTP-backed module
func (m *Module) Initialize(host service.Host) error {
	return nil
}

// Start begins serving requests over HTTP
func (m *Module) Start(ready chan<- struct{}) <-chan error {
	m.mux = http.NewServeMux()
	// Do something unrelated to annotations
	m.router = mux.NewRouter()

	m.mux.Handle("/", m.router)

	healthFound := false
	for _, h := range m.handlers {
		if h.Path == healthPath {
			healthFound = true
		}
		handle := h.Handler
		handle = panicWrap(handle)
		// TODO other middlewares, logging, tracing?
		route := m.router.Handle(h.Path, handle)
		// apply all route options
		for _, opt := range h.Options {
			opt(Route{route})
		}
	}

	if !healthFound {
		m.router.HandleFunc(healthPath, handleHealth)
	}

	// Debug is opt-out
	if m.config.Debug == nil || *m.config.Debug {
		m.router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
	}

	ret := make(chan error, 1)

	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		ret <- err
		return ret
	}

	// finally, start the http server.
	// TODO update log object to be accessed via http context #74
	m.log.Info("Server listening on port", "port", m.config.Port)

	if err != nil {
		ret <- err
		return ret
	}
	m.listenMu.Lock()
	m.listener = listener
	m.listenMu.Unlock()

	go func() {
		listener := m.accessListener()
		ready <- struct{}{}
		ret <- nil
		if err := http.Serve(listener, m.mux); err != nil {
			m.log.Error("HTTP Serve error", "error", err)
		}
	}()
	return ret
}

// Stop shuts down an HTTP module
func (m *Module) Stop() error {
	m.listenMu.Lock()
	defer m.listenMu.Unlock()

	var err error
	if m.listener != nil {
		err = m.listener.Close()
		m.listener = nil
	}
	return err
}

// Thread-safe access to the listener object
func (m *Module) accessListener() net.Listener {
	m.listenMu.RLock()
	defer m.listenMu.RUnlock()

	return m.listener
}

// IsRunning returns whether the module is currently running
func (m *Module) IsRunning() bool {
	return m.accessListener() != nil
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
				fmt.Fprintf(w, "Server error: %+v", err)
			}
		}()

		h.ServeHTTP(w, r)
	}
}
