package decorator

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Layer represents the method call between two service layers.
type Layer func(context.Context, ...interface{}) (interface{}, error)

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

type plainUnaryWrap struct {
	layer Layer
}

// UnaryWrap ...
func UnaryWrap(layer Layer) transport.UnaryHandler {
	return &plainUnaryWrap{
		layer: layer,
	}
}

func (p plainUnaryWrap) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	_, err := p.layer(ctx, req, resw)
	return err
}

type plainOnewayWrap struct {
	layer Layer
}

// OnewayWrap ...
func OnewayWrap(layer Layer) transport.OnewayHandler {
	return &plainOnewayWrap{
		layer: layer,
	}
}

func (p plainOnewayWrap) HandleOneway(ctx context.Context, req *transport.Request) error {
	_, err := p.layer(ctx, req)
	return err
}
