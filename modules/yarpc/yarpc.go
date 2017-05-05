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

type Module struct {
	l           *zap.Logger
	handlerCtor fx.Component
	d           *yarpc.Dispatcher
}

type Transports struct {
	Ts []transport.Procedure
}

type starter struct{}

func New(handlerCtor fx.Component) *Module {
	if handlerCtor == nil {
		panic("Expect a non nil handler constructor")
	}

	return &Module{handlerCtor: handlerCtor}
}

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

func (m *Module) Stop() error {
	if err := m.d.Stop(); err != nil {
		m.l.Error("Failed to stop dispatcher", zap.Error(err))
		return err
	}

	m.l.Info("Dispatcher stopped")
	return nil
}
