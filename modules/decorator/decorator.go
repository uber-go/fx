package decorator

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Layer represents the method call between two service layers.
type Layer func(context.Context, ...interface{}) error

// Decorator is a chainable behavior modifier for layer handlers.
type Decorator func(Layer) Layer

// Build wraps the provided layer with decorators
func Build(l Layer, m ...Decorator) Layer {
	layer := l
	for i := len(m) - 1; i >= 0; i-- {
		layer = m[i](layer)
	}
	return layer
}

type unaryWrap struct {
	layer Layer
}

// UnaryWrap ...
func UnaryWrap(layer Layer) transport.UnaryHandler {
	return &unaryWrap{
		layer: layer,
	}
}

func (p unaryWrap) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return p.layer(ctx, req, resw)
}

// LayerWrap ...
func LayerWrap(handler transport.UnaryHandler) func(ctx context.Context, req ...interface{}) error {
	return func(ctx context.Context, req ...interface{}) error {
		return handler.Handle(ctx, req[0].(*transport.Request), req[1].(transport.ResponseWriter))
	}
}

type onewayWrap struct {
	layer Layer
}

// OnewayWrap ...
func OnewayWrap(layer Layer) transport.OnewayHandler {
	return &onewayWrap{
		layer: layer,
	}
}

func (p onewayWrap) HandleOneway(ctx context.Context, req *transport.Request) error {
	return p.layer(ctx, req)
}
