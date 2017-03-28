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

package yarpc

import (
	"go.uber.org/fx/config"
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"

	"go.uber.org/zap"
)

const (
	recovery    = "recovery"
	breaker     = "breaker"
	logger      = "logger"
	rateLimiter = "ratelimiter"
	retry       = "retry"
	timeout     = "timeout"
)

// InboundUnaryMiddlewareChainBuilder keeps all the procedures to build
type InboundUnaryMiddlewareChainBuilder struct {
	host       service.Host
	procedures map[string][]decorator.UnaryDecorator
}

// populate from config for creating decorator per procedure
type middlewareConfig struct {
	Procedures map[string]decoratorConfig
}

type decoratorConfig struct {
	Middleware []string
}

// Compile compiles all the Decorators for the TransportUnaryMiddleware
func (ch InboundUnaryMiddlewareChainBuilder) Compile() {
	var cfg middlewareConfig
	if err := ch.host.Config().Get("modules").Get(ch.host.ModuleName()).Populate(&cfg); err != nil {
		zap.L().Warn("can't read middleware config")
	}

	scopedCfg := config.NewScopedProvider("middleware", ch.host.Config())
	for procedure, decorators := range cfg.Procedures {
		for _, d := range decorators.Middleware {
			switch d {
			case recovery:
				dec := decorator.Recovery(ch.host.Metrics(), config.NewScopedProvider(recovery, scopedCfg))
				ch.procedures[procedure] = append(ch.procedures[recovery], dec)
			}
		}
	}
}
