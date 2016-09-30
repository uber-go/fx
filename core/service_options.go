// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
