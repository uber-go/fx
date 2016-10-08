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
	"reflect"

	"github.com/uber-go/uberfx/core/config"
	cm "github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/uberfx/internal/util"
)

// A ServiceState represents the state of a service
type ServiceState int

const (
	// Uninitialized means a service has not yet been initialized
	Uninitialized = iota + 1
	// Initialized means a service has been initialized
	Initialized
	// Starting represents a service in the process of starting
	Starting
	// Running represents a running service
	Running
	// Stopping represents a service in the process of stopping
	Stopping
	// Stopped represents a service that has been shut down
	Stopped
)

// A ServiceOwner encapsulates service ownership
type ServiceOwner interface {
	ServiceHost
	Start(waitForExit bool) (<-chan ServiceExit, error)
	Stop(reason string, exitCode int) error
}

// ServiceInstance is the interface that is implemented by user service/
// code.
type ServiceInstance interface {

	// OnInit will be called after the service has been initialized
	OnInit(service ServiceHost) error

	// OnStateChange is called whenever the service changes
	// states
	OnStateChange(old ServiceState, new ServiceState)

	// OnShutdown is called before the service shuts down
	OnShutdown(reason ServiceExit)

	// OnCriticalError is called in response to a critical error.  If false
	// is returned the service will shut down, otherwise the error will be ignored.
	OnCriticalError(err error) bool
}

// ServiceExit is a signal for a service that needs to exit
type ServiceExit struct {
	Reason   string
	Error    error
	ExitCode int
}

type serviceConfig struct {
	ServiceName        string   `yaml:"applicationID" required:"true"`
	ServiceOwner       string   `yaml:"applicationOwner"  required:"true"`
	ServiceDescription string   `yaml:"applicationDesc"`
	ServiceRoles       []string `yaml:"roles"`
}

// NewService creates a service owner from a set of service instances and
// options
func NewService(instance ServiceInstance, options ...ServiceOption) ServiceOwner {
	cfg := config.Global()

	svc := &serviceHost{
		// TODO: get these out of config struct instead
		modules: []Module{},
		serviceCore: serviceCore{
			configProvider: cfg,
			instance:       instance,
			items:          map[string]interface{}{},
		},
	}

	// prepend the default options.
	// TODO: This isn't right.  Do we order the options so we make sure to use
	// the passed in options?
	//
	WithConfiguration(cfg)(svc)

	// load standard config
	svc.configProvider.GetValue("").PopulateStruct(&svc.standardConfig)

	// hash up the roles
	svc.roles = map[string]bool{}
	for _, r := range svc.standardConfig.ServiceRoles {
		svc.roles[r] = true
	}

	WithMetricsScope(cm.Global(true))(svc)

	// Run the rest of the options
	for _, opt := range options {
		if optionErr := opt(svc); optionErr != nil {
			panic(optionErr)
		}
	}

	// if we have an instance, look for a property called "config" and load the service
	// node into that config.
	if instance != nil {

		loadInstanceConfig(svc.configProvider, "service", instance)

		if field, found := util.FindField(instance, nil, reflect.TypeOf((ServiceHost)(nil))); found {
			var sc ServiceHost = &svc.serviceCore
			field.Set(reflect.ValueOf(sc))
		}
		svc.instance = instance
	}

	svc.Metrics().Counter("boot").Inc(1)
	return svc
}
