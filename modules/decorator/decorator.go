package decorator

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Layer represents the method call between two service layers.
type UnaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

func (u UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return u(ctx, req, resw)
}

// Decorator is a chainable behavior modifier for layer handlers.
type Decorator func(UnaryHandlerFunc) UnaryHandlerFunc

// Build wraps the provided layer with decorators
func Build(l UnaryHandlerFunc, m ...Decorator) UnaryHandlerFunc {
	layer := l
	for i := len(m) - 1; i >= 0; i-- {
		layer = m[i](layer)
	}
	return layer
}
