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
	"fmt"
	"net"
	"net/http"

	"go.uber.org/fx"
)

// --8<-- [start:provide-server-1]
func main() {
	// --8<-- [start:app]
	fx.New(
		fx.Provide(NewHTTPServer),
		// --8<-- [end:provide-server-1]
		fx.Invoke(func(*http.Server) {}),
	// --8<-- [start:provide-server-2]
	).Run()
	// --8<-- [end:app]
}

// --8<-- [end:provide-server-2]

// --8<-- [start:partial-1]

// NewHTTPServer builds an HTTP server that will begin serving requests
// when the Fx application starts.
// --8<-- [start:full]
func NewHTTPServer(lc fx.Lifecycle) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	// --8<-- [end:partial-1]
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	// --8<-- [start:partial-2]
	return srv
}

// --8<-- [end:partial-2]
// --8<-- [end:full]
