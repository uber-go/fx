package main

import (
	"fmt"

	"github.com/uber-go/uberfx/core"
)

type ServiceApp struct {
	core.Service
	Config serviceConfig
}

var _ core.ServiceInstance = &ServiceApp{}

func (service *ServiceApp) OnInit(svc *core.Service) error {
	fmt.Printf("The config value is %v\n", service.Config.SomeNumber)
	return nil
}

func (service *ServiceApp) OnShutdown(reason core.ServiceExit) {
}

func (service *ServiceApp) OnCriticalError(err error) bool {
	return false
}
