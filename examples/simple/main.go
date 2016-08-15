package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules/http"
)

func main() {

	service := core.NewService(
		&MyService{},
		core.WithModules(http.Module()),
	)
	service.Start(true)
}
