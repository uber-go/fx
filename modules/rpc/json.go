package rpc

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules"

	"github.com/yarpc/yarpc-go/encoding/json"
)

type CreateJsonRegistrantsFunc func(service core.ServiceHost) []json.Registrant

func JsonModule(hookup CreateJsonRegistrantsFunc, options ...modules.ModuleOption) core.ModuleCreateFunc {
	return func(mi core.ModuleCreateInfo) ([]core.Module, error) {
		if mod, err := newYarpcJsonModule(mi, hookup, options...); err == nil {
			return []core.Module{mod}, nil
		} else {
			return nil, err
		}

	}
}

func newYarpcJsonModule(mi core.ModuleCreateInfo, createService CreateJsonRegistrantsFunc, options ...modules.ModuleOption) (*YarpcModule, error) {

	reg := func(mod *YarpcModule) {
		procs := createService(mi.Host)

		if procs != nil {
			for _, proc := range procs {
				json.Register(mod.rpc, proc)
			}
		}
	}

	return newYarpcModule(mi, reg, options...)
}
