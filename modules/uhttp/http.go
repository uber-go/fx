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
	"time"

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
	mcb      inboundMiddlewareChainBuilder
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
func New(hookup GetHandlersFunc) service.ModuleCreateFunc {
	return func(mi service.ModuleInfo) (service.Module, error) {
		return newModule(mi, hookup)
	}
}

func newModule(
	mi service.ModuleInfo,
	getHandlers GetHandlersFunc,
) (*Module, error) {
	// setup config defaults
	cfg := Config{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}
	if err := mi.Config().Scope("modules").Get(mi.Name()).PopulateStruct(&cfg); err != nil {
		mi.Logger(context.Background()).Error("Error loading http module configuration", "error", err)
	}
	module := &Module{
		ModuleInfo: mi,
		handlers:   addHealth(getHandlers(mi)),
		// TODO(pedge): issue with module name here, we will register this logger
		// before any naming overrides can happen in the service package
		mcb:    defaultInboundMiddlewareChainBuilder(mi.Logger(context.Background()), mi.AuthClient()),
		config: cfg,
	}
	stats.SetupHTTPMetrics(mi.Metrics())
	middleware := inboundMiddlewareFromModuleInfo(mi)
	module.mcb = module.mcb.AddMiddleware(middleware...)
	return module, nil
}

// Name returns the default name
func (m *Module) Name() string {
	return "http"
}

// Start begins serving requests over HTTP
func (m *Module) Start() error {
	mux := http.NewServeMux()
	// Do something unrelated to annotations
	router := NewRouter(m.ModuleInfo)

	mux.Handle("/", router)

	for _, h := range m.handlers {
		router.Handle(h.Path, m.mcb.Build(h.Handler))
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
	m.Logger(context.Background()).Info("Server listening on port", "port", m.config.Port)

	m.listener = listener
	m.srv = &http.Server{Handler: mux}
	go func() {
		// TODO(pedge): what to do about error?
		if err := m.srv.Serve(m.listener); err != nil {
			m.Logger(context.Background()).Error("HTTP Serve error", "error", err)
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
