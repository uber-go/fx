package modules

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/metrics"
)

type ModuleConfig struct {
	Roles []string `yaml:"roles"`
}

type ModuleBase struct {
	moduleType string
	name       string
	service    *core.Service
	isRunning  bool
	reporter   metrics.TrafficReporter
	roles      []string
}

func NewModuleBase(moduleType string, name string, service *core.Service, reporter metrics.TrafficReporter, roles []string) *ModuleBase {
	return &ModuleBase{
		moduleType: moduleType,
		name:       name,
		service:    service,
		reporter:   reporter,
		roles:      roles,
	}
}

func (mb ModuleBase) Roles() []string {
	return mb.roles
}
func (mb ModuleBase) Type() string {
	return mb.moduleType
}

func (mb ModuleBase) Name() string {
	return mb.name
}

func (mb ModuleBase) Reporter() metrics.TrafficReporter {
	return mb.reporter
}
