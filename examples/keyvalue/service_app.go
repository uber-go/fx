package main

import (
	"fmt"

	"github.com/uber-go/uberfx/core"
)

// Define your service instance
type MyService struct {
	core.ServiceHost
	ServiceConfig serviceConfig
	someFlag      bool
}

// These will be called for doing tasks at init and shutdown

func (service *MyService) OnInit(svc core.ServiceHost) error {
	fmt.Printf("The config value for %q is %v\n", service.Name(), service.ServiceConfig.SomeNumber)

	return nil
}

func (service *MyService) OnStateChange(old core.ServiceState, new core.ServiceState) {

}

func (service *MyService) OnShutdown(reason core.ServiceExit) {
}

func (service *MyService) OnCriticalError(err error) bool {
	return false
}

var _ core.ServiceInstance = &MyService{}
