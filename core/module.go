package core

import "github.com/uber-go/uberfx/core/metrics"

type ModuleType string

type Module interface {
	Initialize(service *Service) error
	Type() string
	Name() string
	Start() chan error
	Stop() error
	IsRunning() bool
	Reporter() metrics.TrafficReporter
	Roles() []string
}
