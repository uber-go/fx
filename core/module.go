package core

import "github.com/uber-go/uberfx/core/metrics"

type ModuleType string

type Module interface {
	Type() string
	Name() string
	Initialize(service *Service) error
	Start() chan error
	Stop() error
	IsRunning() bool
	Reporter() metrics.TrafficReporter
	Roles() []string
}

type ModuleBase struct {
	moduleType string
	name       string
	service    *Service
	isRunning  bool
	reporter   metrics.TrafficReporter
	roles      []string
}

type ModuleConfig struct {
	Roles []string `yaml:"roles"`
}

func NewModuleBase(moduleType string, name string, service *Service, reporter metrics.TrafficReporter, roles []string) *ModuleBase {
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
