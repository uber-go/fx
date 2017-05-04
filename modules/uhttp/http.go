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

	"go.uber.org/fx"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"

	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	"go.uber.org/zap"
)

const (
	// default healthcheck endpoint
	healthPath = "/health"

	// default pprof endpoint
	pprofPath = "/debug/pprof"
)

// A Module is a module to handle HTTP requests
type Module struct {
	listener    net.Listener
	lock        sync.RWMutex
	l           *zap.Logger
	handlerCtor fx.Component
}

// Config handles config for HTTP modules
type Config struct {
	Port    int           `yaml:"port" default:"3001"`
	Timeout time.Duration `yaml:"timeout" default:"60s"`
	Debug   bool          `yaml:"debug" default:"true"`
}

// New returns a new HTTP ModuleProvider.
func New(handlerCtor fx.Component) *Module {
	if handlerCtor == nil {
		panic("Handler constructor is nil")
	}

	return &Module{handlerCtor: handlerCtor}
}

// Name returns uhttp.
func (m *Module) Name() string {
	return "uhttp"
}

type starter struct{}

// Constructor returns module components: handler constructor and harness for it.
func (m *Module) Constructor() []fx.Component {
	return []fx.Component{
		m.handlerCtor,
		func(provider config.Provider, l *zap.Logger, scope tally.Scope, handler http.Handler) (*starter, error) {
			// setup config defaults
			cfg := Config{}

			m.l = ulog.Logger(context.Background()).With(zap.String("module", m.Name()))
			if err := provider.Get("modules").Get(m.Name()).Populate(&cfg); err != nil {
				m.l.Error("Error loading http module configuration", zap.Error(err))
			}

			serveMux := http.NewServeMux()
			serveMux.Handle(healthPath, healthHandler{})

			// TODO: pass in the auth client as part of module construction
			authClient := auth.Load(provider, scope)
			stats := newStatsClient(scope)

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
			m.listener = listener

			// finally, start the http server.
			m.l.Info("Server listening on port", zap.Int("port", cfg.Port))
			srv := &http.Server{
				Handler: serveMux,
			}

			go func() {
				if err := srv.Serve(listener); err != nil {
					m.l.Error("HTTP Serve error", zap.Error(err))
				}
			}()

			return &starter{}, nil
		},
	}
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
