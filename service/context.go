package service

import gcontext "context"

// Context embeds Host and go context for use
type Context interface {
	gcontext.Context
	Host
}
type context struct {
	gcontext.Context
	Host

	resources interface{}
}

// NewContext always returns service.Context for use in the service
func NewContext(ctx gcontext.Context, host Host, resources interface{}) Context {
	return &context{
		Context:   ctx,
		Host:      host,
		resources: resources,
	}
}

func (c *context) Resources() interface{} {
	return c.resources
}
