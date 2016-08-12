package main

import (
	"fmt"

	"github.com/uber-go/uberfx/core"
)

type Service struct {
	core.Service
	Config serviceConfig
}

var _ core.ServiceInstance = &Service{}

func (service *Service) OnInit(svc *core.Service) error {
	fmt.Println(service.Name())
	return nil
}

func (service *Service) OnShutdown(reason core.ServiceExit) {
}

func (service *Service) OnCriticalError(err error) bool {
	return false
}
