package main

import (
	"fmt"

	"github.com/uber-go/uberfx/core"
)

// Define your service instance
type MyService struct {
	core.Service
	Config   serviceConfig
	someFlag bool
}

// These will be called for doing tasks at init and shutdown

func (service *MyService) OnInit(svc *core.Service) error {
	fmt.Printf("The config value is %v\n", service.Config.SomeNumber)
	return nil
}

func (service *MyService) OnShutdown(reason core.ServiceExit) {
}

func (service *MyService) OnCriticalError(err error) bool {
	return false
}
