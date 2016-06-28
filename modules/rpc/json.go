package rpc

import (
	"github.com/uber-go/uberfx/core"
	"github.com/yarpc/yarpc-go/encoding/json"
)

type CreateJsonRegistrantsFunc func(service *core.Service) []json.Registrant

func JsonModule(name string, moduleRoles []string, hookup CreateJsonRegistrantsFunc) core.ModuleCreateFunc {
	return func(svc *core.Service) ([]core.Module, error) {
		if mod, err := newYarpcJsonModule(name, svc, moduleRoles, hookup); err != nil {
			return nil, err
		} else {
			return []core.Module{mod}, nil
		}
	}
}

func newYarpcJsonModule(name string, service *core.Service, roles []string, createService CreateJsonRegistrantsFunc) (*YarpcModule, error) {

	reg := func(mod *YarpcModule) {
		procs := createService(service)

		if procs != nil {
			for _, proc := range procs {
				json.Register(mod.rpc, proc)
			}
		}
	}

	return newYarpcModule(name, service, roles, reg)
}
