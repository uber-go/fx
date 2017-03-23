package yarpc

import (
	"fmt"

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
	Procedures map[string]layerConfig
}

type layerConfig struct {
	Layers []string
}

// Compile compiles all the Decorators for the TransportUnaryMiddleware
func (ch InboundUnaryMiddlewareChainBuilder) Compile() {
	var cfg middlewareConfig
	if err := ch.host.Config().Get("modules").Get(ch.host.ModuleName()).Get("middleware").Populate(&cfg); err != nil {
		fmt.Println("can't read middleware config")
	}

	scopedCfg := config.NewScopedProvider("middleware", ch.host.Config())
	for procedure, layers := range cfg.Procedures {
		for _, layer := range layers.Layers {
			switch layer {
			case recovery:
				decorator := decorator.Recovery(ch.host.Metrics(), config.NewScopedProvider(recovery, scopedCfg))
				ch.procedures[procedure] = append(ch.procedures[recovery], decorator)
			}
		}
	}
}
