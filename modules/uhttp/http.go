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
	"net"
	"net/http"
	_ "net/http/pprof" // for automatic pprof
	"sync"
	"time"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/modules/uhttp/internal/stats"
	"go.uber.org/fx/service"

	"github.com/pkg/errors"
)

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

// A Module is a module to handle HTTP requests
type Module struct {
	service.ModuleInfo
	config   Config
	srv      *http.Server
	listener net.Listener
	handlers []RouteHandler
	listenMu sync.RWMutex
	fcb      filterChainBuilder
}

var _ service.Module = &Module{}

// Config handles config for HTTP modules
type Config struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
	Debug   *bool         `yaml:"debug"`
}

// GetHandlersFunc returns a slice of registrants from a service host
type GetHandlersFunc func(service service.Host) []RouteHandler

// New returns a new HTTP module
func New(hookup GetHandlersFunc, filters []Filter) service.ModuleCreateFunc {
	return func(mi service.ModuleInfo) (service.Module, error) {
		return newModule(mi, hookup, filters)
	}
}

func newModule(
	mi service.ModuleInfo,
	getHandlers GetHandlersFunc,
	filters []Filter,
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

	stats.SetupHTTPMetrics(mi.Host.Metrics())

	handlers := addHealth(getHandlers(mi.Host))

	// TODO (madhu): Add other middleware - logging, metrics.
	module := &Module{
		ModuleBase: *modules.NewModuleBase(mi.Name, mi.Host, []string{}),
		handlers:   handlers,
		fcb:        defaultFilterChainBuilder(mi.Host),
	}

	module.fcb = module.fcb.AddFilters(filters...)

	err := module.Host().Config().Get(getConfigKey(mi.Name)).PopulateStruct(cfg)
	if err != nil {
		module.Host().Logger().Error("Error loading http module configuration", "error", err)
	}
	module.config = *cfg

	module.log = module.Host().Logger().With("moduleName", mi.Name)

	for _, option := range options {
		if err := option(&mi); err != nil {
			module.log.Error("Unable to apply option", "error", err, "option", option)
			return module, errors.Wrap(err, "unable to apply option to module")
		}
	}

	return module, nil
}

// Start begins serving requests over HTTP
func (m *Module) Start() error {
	mux := http.NewServeMux()
	// Do something unrelated to annotations
	router := NewRouter(m.Host())

	mux.Handle("/", router)

	for _, h := range m.handlers {
		router.Handle(h.Path, m.fcb.Build(h.Handler))
	}

	if m.config.Debug == nil || *m.config.Debug {
		router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
	}

	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		return errors.Wrap(err, "unable to open TCP listener for HTTP module")
	}
	// finally, start the http server.
	// TODO update log object to be accessed via http context #74
	m.log.Info("Server listening on port", "port", m.config.Port)

	m.listener = listener
	m.srv = &http.Server{Handler: mux}
	go func() {
		if err := m.srv.Serve(m.listener); err != nil {
			m.Logger().Error("HTTP Serve error", "error", err)
		}
	}()
	return nil
}

// Stop shuts down an HTTP module
func (m *Module) Stop() error {
	var err error
	if m.listener != nil {
		// TODO: Change to use https://tip.golang.org/pkg/net/http/#Server.Shutdown
		// once we upgrade to Go 1.8
		// GFM-258
		err = m.listener.Close()
		m.listener = nil
	}
	return err
}

// addHealth adds in the default if health handler is not set
func addHealth(handlers []RouteHandler) []RouteHandler {
	healthFound := false
	for _, h := range handlers {
		if h.Path == healthPath {
			healthFound = true
		}
	}
	if !healthFound {
		handlers = append(handlers, NewRouteHandler(healthPath, healthHandler{}))
	}
	return handlers
}
