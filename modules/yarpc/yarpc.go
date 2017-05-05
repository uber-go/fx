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
	"go.uber.org/fx/config"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	yconfig "go.uber.org/yarpc/x/config"

	"github.com/uber-go/tally"
	"go.uber.org/zap"
)

// Module represents a collection of YARPC components.
type Module struct {
	l           *zap.Logger
	handlerCtor fx.Component
	d           *yarpc.Dispatcher
}

// Transports represent a collection Procedures that will be registered with a dispatcher.
type Transports struct {
	//Ts - transports to be registered.
	Ts []transport.Procedure
}

type starter struct{}

// New creates a new YARPC Module with a handler constructor.
func New(handlerCtor fx.Component) *Module {
	if handlerCtor == nil {
		panic("Expect a non nil handler constructor")
	}

	return &Module{handlerCtor: handlerCtor}
}

// Name returns yarpc
func (m *Module) Name() string {
	return "yarpc"
}

func (m *Module) populateConfig(provider config.Provider) (yarpc.Config, error) {
	var cfg = yconfig.New()
	cfg.MustRegisterTransport(http.TransportSpec())
	cfg.MustRegisterTransport(tchannel.TransportSpec())
	val := provider.Get("modules").Get(m.Name()).Value()
	return cfg.LoadConfig(provider.Get("name").AsString(), val)
}

// Constructor returns Module components.
func (m *Module) Constructor() []fx.Component {
	return []fx.Component{
		func(provider config.Provider, scope tally.Scope, logger *zap.Logger) (*yarpc.Dispatcher, error) {
			m.l = logger.With(zap.String("module", m.Name()))
			c, err := m.populateConfig(provider)
			if err != nil {
				m.l.Error("Failed to populate config", zap.Error(err))
				return nil, err
			}

			m.d = yarpc.NewDispatcher(c)
			return m.d, nil
		},
		m.handlerCtor,
		func(transports *Transports) (*starter, error) {
			m.d.Register(transports.Ts)
			if err := m.d.Start(); err != nil {
				m.l.Error("Failed to start dispatcher", zap.Error(err))
				return nil, err
			}

			m.l.Info("Dispatcher started successfully")
			return &starter{}, nil
		},
	}
}

// Stop stops the dispatcher.
func (m *Module) Stop() error {
	if err := m.d.Stop(); err != nil {
		m.l.Error("Failed to stop dispatcher", zap.Error(err))
		return err
	}

	m.l.Info("Dispatcher stopped")
	return nil
}
