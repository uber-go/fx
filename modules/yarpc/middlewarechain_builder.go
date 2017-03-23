package yarpc

import (
	"go.uber.org/fx/config"
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"
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
	procedures map[string][]decorator.Decorator
}

type InboundUnaryMiddlewareChain struct {
	host   service.Host
	layers decorator.Layer
}

// populate from config for creating decorator per procedure
type middlewareConfig struct {
	procedures map[string]interface{}
}

// Compile compiles all the Decorators for the TransportUnaryMiddleware
func (ch InboundUnaryMiddlewareChainBuilder) Compile() {
	var m middlewareConfig
	scopedCfg := config.NewScopedProvider("middleware", ch.host.Config())
	ch.host.Config().Get("middleware").Populate(&m)
	for procedure := range m.procedures {
		switch procedure {
		case recovery:
			decorator := decorator.Recovery(ch.host.Metrics(), config.NewScopedProvider(recovery, scopedCfg))
			ch.procedures[recovery] = append(ch.procedures[recovery], decorator)
		}
	}
}
