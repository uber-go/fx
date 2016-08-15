package core

import "github.com/uber-go/uberfx/core/metrics"

type ModuleType string

type Module interface {
	Initialize(host ServiceHost) error
	Type() string
	Name() string
	Start() <-chan error
	Stop() error
	IsRunning() bool
	Reporter() metrics.TrafficReporter
}

type ModuleCreateInfo struct {
	Name  string
	Roles []string
	Items map[string]interface{}
	Host  ServiceHost
}

type ModuleCreateFunc func(ModuleCreateInfo) ([]Module, error)
