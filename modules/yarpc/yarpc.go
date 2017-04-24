// Copyright (c) 2017 Uber Technologies, Inc.
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

package yarpc

import (
	"go.uber.org/fx"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

// Module foo
type Module struct{}

// New foo
func New() *Module {
	return &Module{}
}

// Name foo
func (m *Module) Name() string {
	return "yarpc"
}

// Transports foo
type Transports struct {
	Ts []transport.Procedure
}

// TODO: figure out the starter
type starter struct{}

// FIXME: dig work-around. Returning just error is not doabled
type foo struct{}

// Constructor foo
func (m *Module) Constructor() fx.Component {
	return func(d *yarpc.Dispatcher, t *Transports) (*foo, error) {
		d.Register(t.Ts)
		return nil, nil
	}
}

// Stop foo
func (m *Module) Stop() {
	panic("ah")
}
