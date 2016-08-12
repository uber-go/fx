package core

import (
	"github.com/uber-go/uberfx/core/config"
	"github.com/uber-go/uberfx/core/metrics"
)

type ServiceOption func(*Service) error

func WithModules(modules ...ModuleCreateFunc) ServiceOption {
	return func(svc *Service) error {
		for _, mcf := range modules {
			var err error
			if mods, err := mcf(svc); err != nil {
				for _, mod := range mods {
					err = svc.addModule(mod)
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
	return func(svc *Service) error {
		svc.configProvider = config
		return nil
	}
}

func WithMetricsScope(scope xm.Scope) ServiceOption {
	return func(svc *Service) error {
		svc.scope = metrics.Global(true)
		return nil
	}
}
