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
	"net"
	"net/http"
	_ "net/http/pprof" // for automatic pprof
	"sync"
	"time"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	service.Host
	config     Config
	authClient auth.Client
	stats      *statsClient
	inbound    http.Handler
	log        *zap.Logger
	srv        *http.Server
	listener   net.Listener
	handlers   []RouteHandler
	lock       sync.RWMutex
}

var _ service.Module = &Module{}

// Config handles config for HTTP modules
type Config struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
	Debug   bool          `yaml:"debug" default:"true"`
}

// GetHandlersFunc returns a slice of registrants from a service host
type GetHandlersFunc func(service service.Host) []RouteHandler

// New returns a new HTTP ModuleProvider.
func New(hookup GetHandlersFunc, options ...ModuleOption) service.ModuleProvider {
	return service.ModuleProviderFromFunc("uhttp", func(host service.Host) (service.Module, error) {
		return newModule(host, hookup, options...)
	})
}

func newModule(
	host service.Host,
	getHandlers GetHandlersFunc,
	options ...ModuleOption,
) (*Module, error) {
	moduleOptions := &moduleOptions{}
	for _, option := range options {
		if err := option(moduleOptions); err != nil {
			return nil, err
		}
	}
	// setup config defaults
	cfg := Config{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}
	log := ulog.Logger(context.Background()).With(zap.String("module", host.ModuleName()))
	if err := host.Config().Get("modules").Get(host.ModuleName()).Populate(&cfg); err != nil {
		log.Error("Error loading http module configuration", zap.Error(err))
	}
	module := &Module{
		Host:       host,
		handlers:   addHealth(getHandlers(host)),
		authClient: host.AuthClient(),
		stats:      newStatsClient(host.Metrics()),
		config:     cfg,
		log:        log,
	}
	return module, nil
}

// Start begins serving requests over HTTP
func (m *Module) Start() error {
	// Do something unrelated to annotations
	router := mux.NewRouter()

	serveMux := http.NewServeMux()
	serveMux.Handle("/", router)

	for _, h := range m.handlers {
		handle := contextInbound(
			panicInbound(
				metricsInbound(
					tracingInbound(
						authorizationInbound(h, m.authClient, m.stats),
					), m.stats,
				), m.stats,
			), m.log)
		router.Handle(h.Path, handle)
	}

	if m.config.Debug {
		router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
	}

	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		return errors.Wrap(err, "unable to open TCP listener for HTTP module")
	}
	// finally, start the http server.
	// TODO update log object to be accessed via http context #74
	m.log.Info("Server listening on port", zap.Int("port", m.config.Port))

	m.listener = listener
	m.srv = &http.Server{Handler: serveMux}
	go func() {
		// TODO(pedge): what to do about error?
		if err := m.srv.Serve(listener); err != nil {
			m.log.Error("HTTP Serve error", zap.Error(err))
		}
	}()
	return nil
}

// Stop shuts down an HTTP module
func (m *Module) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()
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
