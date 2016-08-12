package modules

import (
	"reflect"

	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/util"
)

type ModuleOption func(core.Module) error

func WithName(name string) ModuleOption {
	return func(module core.Module) error {
		if mb := findModuleInfo(module); mb != nil {
			mb.name = name
		}
		return nil
	}
}

func WithRoles(roles ...string) ModuleOption {
	return func(module core.Module) error {
		if mb := findModuleInfo(module); mb != nil {
			mb.roles = roles
		}
		return nil
	}
}

func findModuleInfo(module core.Module) *ModuleBase {
	if val, ok := util.FindField(module, nil, reflect.TypeOf(ModuleBase{})); ok {
		return val.Interface().(*ModuleBase)
	}
	return nil
}
