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
	"go.uber.org/fx/core/ulog"
)

// A ServiceState represents the state of a service
type ServiceState int

const (
	// Uninitialized means a service has not yet been initialized
	Uninitialized = ServiceState(iota + 1)
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
	Start(waitForExit bool) (exit <-chan ServiceExit, ready <-chan struct{}, err error)
	Stop(reason string, exitCode int) error
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
func NewService(options ...ServiceOption) ServiceOwner {
	cfg := config.Global()

	svc := &serviceHost{
		// TODO: get these out of config struct instead
		modules: []Module{},
		serviceCore: serviceCore{
			configProvider: cfg,
			items:          map[string]interface{}{},
		},
	}

	// prepend the default options.
	// TODO: This isn't right.  Do we order the options so we make sure to use
	// the passed in options?
	//
	ensureThat(WithConfiguration(cfg)(svc), "configuration")

	// load standard config
	svc.configProvider.GetValue("").PopulateStruct(&svc.standardConfig)

	// load and configure logging
	svc.configProvider.GetValue("logging").PopulateStruct(&svc.logConfig)
	ulog.Configure(svc.logConfig)
	ensureThat(WithLogger(ulog.Logger())(svc), "log configuration")

	// hash up the roles
	svc.roles = map[string]bool{}
	for _, r := range svc.standardConfig.ServiceRoles {
		svc.roles[r] = true
	}

	// Run the rest of the options
	for _, opt := range options {
		if optionErr := opt(svc); optionErr != nil {
			panic(optionErr)
		}
	}

	// if we have an observer, look for a property called "config" and load the service
	// node into that config.
	if svc.observer != nil {
		loadInstanceConfig(svc.configProvider, "service", svc.observer)

		if shc, ok := svc.observer.(SetContainerer); ok {
			shc.SetContainer(svc)
		}
	}

	svc.Metrics().Counter("boot").Inc(1)
	return svc
}

func ensureThat(err error, description string) {
	if err != nil {
		ulog.Logger().With("error", err).Fatal("Fatal error initializing app: ", description)
	}
}
