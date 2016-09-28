package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules/rpc"
	"github.com/yarpc/yarpc-go/encoding/json"
)

func registerJSONers(service core.ServiceHost) []json.Registrant {
	return nil
}

func main() {
	service := core.NewService(
		&MyService{},
		core.WithModules(
			rpc.JsonModule(rpc.CreateJsonRegistrantsFunc(registerJSONers)),
		),
	)
	service.Start(true)
}
