package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules"
	"github.com/uber-go/uberfx/modules/rpc"
)

func main() {

	service := core.NewService(
		&MyService{},
		core.WithModules(
			// Create a YARPC module that exposes endpoints
			rpc.ThriftModule(
				rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
				modules.WithRoles("service")),
		),
	)

	service.Start(true)
}
