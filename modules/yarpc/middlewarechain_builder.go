package yarpc

import (
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"
)

type InboundUnaryMiddlewareChainBuilder struct {
	host       service.Host
	procedures map[string][]decorator.Decorator
}

type middlewares struct {
	procs              map[string]interface{}
	defaultMiddlewares []string
}

func (ch InboundUnaryMiddlewareChainBuilder) Compile() {
	// compile procedures with set of decorators
	// layer := decorator.Recovery(ch.host.Metrics())

	// handler = decorator.UnaryWrap(decorator.Build(layer, ch.procedures[req.Procedure]))
}

func Build() map[string][]decorator.Decorator {
	// populate decorators based on the configuration and Construct
}
