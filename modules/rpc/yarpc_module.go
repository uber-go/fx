package rpc

import (
	"fmt"
	"log"

	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/config"
	"github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/uberfx/modules"

	"github.com/uber/tchannel-go"
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"
)

// module

type YarpcModule struct {
	modules.ModuleBase
	rpc      yarpc.Dispatcher
	register registerServiceFunc
	config   RPCConfig
}

var _ core.Module = &YarpcModule{}

type registerServiceFunc func(module *YarpcModule)

const RPCModuleType = "rpc"

type RPCConfig struct {
	modules.ModuleConfig
	Bind          string `yaml:"bind"`
	AdvertiseName string `yaml:"advertise_name"`
}

func newYarpcModule(mi core.ModuleCreateInfo, reg registerServiceFunc, options ...modules.ModuleOption) (*YarpcModule, error) {

	for _, opt := range options {
		opt(mi)
	}

	cfg := &RPCConfig{
		AdvertiseName: mi.Host.Name(),
		Bind:          ":0",
	}

	name := "yarpc"
	if mi.Name != "" {
		name = mi.Name
	}

	reporter := &metrics.LoggingTrafficReporter{Prefix: mi.Host.Name()}

	module := &YarpcModule{
		ModuleBase: *modules.NewModuleBase(RPCModuleType, name, mi.Host, reporter, []string{}),
		register:   reg,
		config:     *cfg,
	}

	if config.Global().GetValue(fmt.Sprintf("modules.%s", module.Name())).PopulateStruct(cfg) {
		// found values, update module
		module.config = *cfg
	}

	return module, nil
}

func (m *YarpcModule) Initialize(service core.ServiceHost) error {
	return nil
}

func (m *YarpcModule) Start() <-chan error {
	channel, err := tchannel.NewChannel(m.config.AdvertiseName, nil)
	if err != nil {
		log.Fatalln(err)
	}

	m.rpc = yarpc.NewDispatcher(yarpc.Config{
		Name: m.config.AdvertiseName,
		Inbounds: []transport.Inbound{
			tch.NewInbound(channel, tch.ListenAddr(m.config.Bind)),
		},
	})

	m.register(m)
	ret := make(chan error, 1)
	log.Printf("Service %q listening on port %v\n", m.config.AdvertiseName, m.config.Bind)

	ret <- m.rpc.Start()
	return ret
}

func (m *YarpcModule) Stop() error {

	// TODO: thread safety
	if m.rpc != nil {
		err := m.rpc.Stop()
		m.rpc = nil
		return err
	}
	return nil
}

func (m *YarpcModule) IsRunning() bool {
	return m.rpc != nil
}
