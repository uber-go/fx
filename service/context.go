// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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

	resources map[string]interface{}
}

// NewContext always returns service.Context for use in the service
func NewContext(ctx gcontext.Context, host Host) Context {
	return &context{
		Context:   ctx,
		Host:      host,
		resources: make(map[string]interface{}),
	}
}

// Resources returns resources associated with the current context
func (c *context) Resource(key string) interface{} {
	if res, ok := c.tryResource(key); ok {
		return res
	}
	return nil
}

func (c *context) tryResource(key string) (interface{}, bool) {
	res, ok := c.resources[key]
	return res, ok
}

// SetResource sets resource on the specified key
func (c *context) SetResource(key string, value interface{}) {
	c.resources[key] = value
}
