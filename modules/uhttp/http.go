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

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"

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

	// default pprof endpoint
	pprofPath = "/debug/pprof"
)

var _ service.Module = &Module{}

// A Module is a module to handle HTTP requests
type Module struct {
	listener net.Listener
	lock     sync.RWMutex
}

// Config handles config for HTTP modules
type Config struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
	Debug   bool          `yaml:"debug" default:"true"`
}

// GetHandlersFunc returns http handler created by caller
type GetHandlersFunc func(service service.Host) http.Handler

// New returns a new HTTP ModuleProvider.
func New(handlerFunc GetHandlersFunc) service.ModuleProvider {
	return service.ModuleProviderFromFunc("uhttp", func(host service.Host) (service.Module, error) {
		handler := handlerFunc(host)
		return newModule(host, handler)
	})
}

func newModule(host service.Host, handler http.Handler) (*Module, error) {
	// setup config defaults
	cfg := Config{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}
	log := ulog.Logger(context.Background()).With(zap.String("module", host.ModuleName()))
	if err := host.Config().Get("modules").Get(host.ModuleName()).Populate(&cfg); err != nil {
		log.Error("Error loading http module configuration", zap.Error(err))
	}

	serveMux := http.NewServeMux()
	serveMux.Handle(healthPath, healthHandler{})

	authClient := host.AuthClient()
	stats := newStatsClient(host.Metrics())

	handle :=
		panicInbound(
			metricsInbound(
				tracingInbound(
					authorizationInbound(handler, authClient, stats),
				), stats,
			), stats,
		)
	serveMux.Handle("/", handle)

	if cfg.Debug {
		serveMux.Handle(pprofPath, http.DefaultServeMux)
	}
	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return nil, errors.Wrap(err, "unable to open TCP listener for HTTP module")
	}

	// finally, start the http server.
	log.Info("Server listening on port", zap.Int("port", cfg.Port))

	srv := &http.Server{
		Handler: serveMux,
	}

	go func() {
		// TODO(pedge): what to do about error?
		if err := srv.Serve(listener); err != nil {
			log.Error("HTTP Serve error", zap.Error(err))
		}
	}()
	return &Module{
		listener: listener,
	}, nil
}

// Start begins serving requests over HTTP
func (m *Module) Start() error {
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
