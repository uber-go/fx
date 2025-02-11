// Copyright (c) 2022 Uber Technologies, Inc.
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

package main

import (
	"context"
	"io"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// --8<-- [start:fx-logger]
func main() {
	fx.New(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		// --8<-- [end:fx-logger]
		// --8<-- [start:provides]
		fx.Provide(
			NewHTTPServer,
			NewServeMux,
			NewEchoHandler,
			zap.NewExample,
		),
		// --8<-- [end:provides]
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

// NewServeMux builds a ServeMux that will route requests
// to the given EchoHandler.
func NewServeMux(echo *EchoHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/echo", echo)
	return mux
}

// EchoHandler is an http.Handler that copies its request body
// back to the response.
// --8<-- [start:echo-init-1]
type EchoHandler struct {
	log *zap.Logger
}

// --8<-- [end:echo-init-1]

// NewEchoHandler builds a new EchoHandler.
// --8<-- [start:echo-init-2]
func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// --8<-- [end:echo-init-2]

// ServeHTTP handles an HTTP request to the /echo endpoint.
// --8<-- [start:echo-serve]
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}
}

// --8<-- [end:echo-serve]

// NewHTTPServer builds an HTTP server that will begin serving requests
// when the Fx application starts.
// --8<-- [start:http-server]
func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{Addr: ":8080", Handler: mux}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
			go srv.Serve(ln)
			// --8<-- [end:http-server]
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
