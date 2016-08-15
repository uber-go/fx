package main

import (
	"fmt"

	"github.com/uber-go/uberfx/core"
)

type MyService struct {
	core.ServiceHost
	ServiceConfig serviceConfig
}

var _ core.ServiceInstance = &MyService{}

func (service *MyService) OnInit(svc core.ServiceHost) error {
	fmt.Println(service.Name())
	return nil
}

func (service *MyService) OnStateChange(old core.ServiceState, new core.ServiceState) {

}

func (service *MyService) OnShutdown(reason core.ServiceExit) {
}

func (service *MyService) OnCriticalError(err error) bool {
	return false
}
