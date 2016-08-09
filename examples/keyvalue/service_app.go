package main

import (
	"github.com/uber-go/uberfx/core"
)

type ServiceApp struct {
	core.Service
}

var _ core.ServiceInstance = &ServiceApp{}

func (service *ServiceApp) OnInit(svc *core.Service) error {
	return nil
}

func (service *ServiceApp) OnShutdown(reason core.ServiceExit) {
}

func (service *ServiceApp) OnCriticalError(err error) bool {
	return false
}
