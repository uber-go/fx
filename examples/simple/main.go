package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules/http"
)

func main() {

	service := core.New(
		&Service{},
		WithModules(http.Module("http", nil)),
	)
	service.Start(true)
}
