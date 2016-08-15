package core

import (
	"github.com/uber-go/uberfx/core/config"
)

type ServiceHost interface {
	Name() string
	Description() string
	Roles() []string
	State() ServiceState
	Metrics() metrics.Scope
	Instance() ServiceInstance
	Config() config.ConfigurationProvider
	Items() map[string]interface{}
}

type serviceCore struct {
	standardConfig serviceConfig
	roles          []string
	state          ServiceState
	configProvider config.ConfigurationProvider
	scope          metrics.Scope
	instance       ServiceInstance
	items          map[string]interface{}
}

var _ ServiceHost = &serviceCore{}

func (s *serviceCore) Name() string {
	return s.standardConfig.ServiceName
}

func (s *serviceCore) Description() string {
	return s.standardConfig.ServiceDescription
}

func (s *serviceCore) Owner() string {
	return s.standardConfig.ServiceOwner
}

func (s *serviceCore) State() ServiceState {
	return s.state
}

func (s *serviceCore) Roles() []string {
	return s.standardConfig.ServiceRoles
}

func (s *serviceCore) Items() map[string]interface{} {
	return s.items
}

func (s *serviceCore) Metrics() metrics.Scope {
	return s.scope
}

func (s *serviceCore) Instance() ServiceInstance {
	return s.instance
}

func (s *serviceCore) Config() config.ConfigurationProvider {
	return s.configProvider
}
