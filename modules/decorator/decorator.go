package decorator

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// UnaryHandlerFunc represents the method call between two service layers.
type UnaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

// Handle adapter for UnaryHandlerFunc
func (u UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return u(ctx, req, resw)
}

// UnaryDecorator is a chainable behavior modifier for layer handlers.
type UnaryDecorator func(UnaryHandlerFunc) UnaryHandlerFunc

// BuildUnary wraps the provided layer with decorators
func BuildUnary(h UnaryHandlerFunc, m ...UnaryDecorator) UnaryHandlerFunc {
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

// UnaryHandlerWrap ...
func UnaryHandlerWrap(handler transport.UnaryHandler) UnaryHandlerFunc {
	return func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		return handler.Handle(ctx, req, resw)
	}
}

// OnewayHandlerFunc represents the method call between two service layers.
type OnewayHandlerFunc func(context.Context, *transport.Request) error

// HandleOneway adapter for OnewayHandlerFunc
func (u OnewayHandlerFunc) HandleOneway(ctx context.Context, req *transport.Request) error {
	return u(ctx, req)
}

// OnewayDecorator is a chainable behavior modifier for layer handlers.
type OnewayDecorator func(OnewayHandlerFunc) OnewayHandlerFunc

// BuildOneway wraps the provided layer with decorators
func BuildOneway(h OnewayHandlerFunc, m ...OnewayDecorator) OnewayHandlerFunc {
	handler := h
	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}
	return handler
}

// OnewayHandlerWrap ...
func OnewayHandlerWrap(handler transport.OnewayHandler) OnewayHandlerFunc {
	return func(ctx context.Context, req *transport.Request) error {
		return handler.HandleOneway(ctx, req)
	}
}
