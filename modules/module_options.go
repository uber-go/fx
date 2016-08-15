package modules

import "github.com/uber-go/uberfx/core"

type ModuleOption func(core.ModuleCreateInfo) error

func WithName(name string) ModuleOption {
	return func(mi core.ModuleCreateInfo) error {
		mi.Name = name
		return nil
	}
}

func WithRoles(roles ...string) ModuleOption {
	return func(mi core.ModuleCreateInfo) error {
		// if mb := findModuleInfo(module); mb != nil {
		// 	mb.roles = roles
		// }
		mi.Roles = roles
		return nil
	}
}

// func findModuleInfo(module core.Module) *ModuleBase {
// 	if val, ok := util.FindField(module, nil, reflect.TypeOf(ModuleBase{})); ok {
// 		mb := reflect.Indirect(val).Interface().(ModuleBase)
// 		return &mb
// 	}
// 	return nil
// }
