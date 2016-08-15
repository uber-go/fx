package core

import (
	"github.com/uber-go/uberfx/core/config"
	"github.com/uber-go/uberfx/core/metrics"
)

type ServiceOption func(ServiceHost) error

// func WithModules(modules ...ModuleInit) ServiceOption {
// 	return func(svc *ServiceHost) error {
// 		for _, mcf := range modules {
// 			var err error
// 			if !svc.supportsRole(mcf.Roles...) {
// 				continue
// 			}
// 			if mods, err := mcf.Factory(svc); err == nil {
// 				for _, mod := range mods {
// 					err = svc.addModule(mod)
// 				}
// 			}
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	}
// }

func WithModules(modules ...ModuleCreateFunc) ServiceOption {
	return func(svc ServiceHost) error {
		svc2 := svc.(*serviceHost)
		for _, mcf := range modules {
			var err error
			mi := ModuleCreateInfo{
				Host:  svc,
				Roles: []string{},
				Items: map[string]interface{}{},
			}

			if mods, err := mcf(mi); err == nil {

				if !svc2.supportsRole(mi.Roles...) {
					continue
				}
				for _, mod := range mods {
					err = svc2.addModule(mod)
				}
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func WithConfiguration(config config.ConfigurationProvider) ServiceOption {
	return func(svc ServiceHost) error {
		svc2 := svc.(*serviceHost)
		svc2.configProvider = config
		return nil
	}
}

func WithMetricsScope(scope xm.Scope) ServiceOption {
	return func(svc ServiceHost) error {
		svc2 := svc.(*serviceHost)
		svc2.scope = metrics.Global(true)
		return nil
	}
}
