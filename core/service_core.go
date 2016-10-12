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
	"go.uber.org/fx/core/config"
)

// A ServiceHost represents the hosting environment for a service instance
type ServiceHost interface {
	Name() string
	Description() string
	Roles() []string
	State() ServiceState
	Metrics() metrics.Scope
	Instance() ServiceInstance
	Config() config.ConfigurationProvider
	Items() map[string]interface{}
}

type serviceCore struct {
	standardConfig serviceConfig
	roles          []string
	state          ServiceState
	configProvider config.ConfigurationProvider
	scope          metrics.Scope
	instance       ServiceInstance
	items          map[string]interface{}
}

var _ ServiceHost = &serviceCore{}

func (s *serviceCore) Name() string {
	return s.standardConfig.ServiceName
}

func (s *serviceCore) Description() string {
	return s.standardConfig.ServiceDescription
}

func (s *serviceCore) Owner() string {
	return s.standardConfig.ServiceOwner
}

func (s *serviceCore) State() ServiceState {
	return s.state
}

func (s *serviceCore) Roles() []string {
	return s.standardConfig.ServiceRoles
}

func (s *serviceCore) Items() map[string]interface{} {
	return s.items
}

func (s *serviceCore) Metrics() metrics.Scope {
	return s.scope
}

func (s *serviceCore) Instance() ServiceInstance {
	return s.instance
}

func (s *serviceCore) Config() config.ConfigurationProvider {
	return s.configProvider
}
