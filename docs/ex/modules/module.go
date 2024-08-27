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

package modules

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module is an example of an Fx module's skeleton.
// --8<-- [start:start]
var Module = fx.Module("server",
	// --8<-- [end:start]
	// --8<-- [start:provide]
	fx.Provide(
		New,
	),
	// --8<-- [end:provide]
	// --8<-- [start:privateProvide]
	fx.Provide(
		fx.Private,
		parseConfig,
	),
	// --8<-- [end:privateProvide]
	// --8<-- [start:invoke]
	fx.Invoke(startServer),
	// --8<-- [end:invoke]
	// --8<-- [start:decorate]
	fx.Decorate(wrapLogger),
	// --8<-- [end:decorate]
// --8<-- [start:endProvide]
)

// --8<-- [end:endProvide]

// Config is the configuration of the server.
// --8<-- [start:config]
type Config struct {
	Addr string `yaml:"addr"`
}

// --8<-- [end:config]

func parseConfig() (Config, error) {
	return Config{}, nil
}

// Params defines the parameters of the module.
// --8<-- [start:params]
type Params struct {
	fx.In

	Log    *zap.Logger
	Config Config
}

// --8<-- [end:params]

// Result defines the results of the module.
// --8<-- [start:result]
type Result struct {
	fx.Out

	Server *Server
}

// --8<-- [end:result]

// New builds a new server.
// --8<-- [start:new]
func New(p Params) (Result, error) {
	// --8<-- [end:new]
	return Result{
		Server: &Server{},
	}, nil
}

// Server is the server.
type Server struct{}

// Start starts the server.
func (*Server) Start() error {
	return nil
}

func startServer(srv *Server) error {
	return srv.Start()
}

func wrapLogger(log *zap.Logger) *zap.Logger {
	return log.With(zap.String("component", "mymodule"))
}
