package yarpc

import "go.uber.org/fx/modules/decorator"

type InboundUnaryMiddleware interface {
	Construct()
}

type inboundUnaryMiddleware struct {
}

func Construct() {
	var recoveryConfig decorator.RecoveryConfig
	// we know the Procedure
	if err := l.host.Config().Get(req.Procedure).Get("middleware").Get("transport").Populate(&recoveryConfig); err != nil {

	}
	layer := decorator.Recovery(l.host, recoveryConfig)(handler)

	handler = decorator.UnaryWrap(decorator.Build(layer, l.procedureMap[req.Procedure]))
}
func Build() map[string][]decorator.Decorator {
	decorators := make(map[string][]decorator.Decorator)
	// populate decorators based on the configuration and Construct
}
