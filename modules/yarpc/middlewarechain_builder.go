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
	Procedures map[string]DecoratorConfig
}

type DecoratorConfig struct {
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
