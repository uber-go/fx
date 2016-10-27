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

package service

import (
	"go.uber.org/fx/core/config"
	"go.uber.org/fx/core/metrics"
	"go.uber.org/fx/core/ulog"

	"github.com/go-validator/validator"
	"github.com/uber-go/tally"
)

// A State represents the state of a service
type State int

const (
	// Uninitialized means a service has not yet been initialized
	Uninitialized = State(iota)
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

// A Owner encapsulates service ownership
type Owner interface {
	Host

	AddModules(modules ...ModuleCreateFunc) error

	Start(waitForExit bool) (exit <-chan Exit, ready <-chan struct{}, err error)
	Stop(reason string, exitCode int) error
}

// Exit is a signal for a service that needs to exit
type Exit struct {
	Reason   string
	Error    error
	ExitCode int
}

type serviceConfig struct {
	ServiceName        string   `yaml:"applicationID" validate:"nonzero"`
	ServiceOwner       string   `yaml:"applicationOwner"  validate:"nonzero"`
	ServiceDescription string   `yaml:"applicationDesc"`
	ServiceRoles       []string `yaml:"roles"`
}

// New creates a service owner from a set of service instances and options
// TODO(glib): Something is fishy here... `service.New` returns a service.Owner -_-
func New(options ...Option) (Owner, error) {
	svc := &host{
		// TODO: get these out of config struct instead
		modules: []Module{},
		serviceCore: serviceCore{
			items: map[string]interface{}{},
		},
	}

	cfg := config.Initialize()
	svc.serviceCore.configProvider = cfg

	// prepend the default options.
	// TODO: This isn't right.  Do we order the options so we make sure to use
	// the passed in options?
	//
	ensureThat(WithConfiguration(cfg)(svc), "configuration")

	// load standard config
	// TODO(glib): `.GetValue("")` is a confusing interface for getting the root config node
	svc.configProvider.GetValue("").PopulateStruct(&svc.standardConfig)

	// load and configure logging
	svc.configProvider.GetValue("logging").PopulateStruct(&svc.logConfig)
	ulog.Configure(svc.logConfig)
	ensureThat(WithLogger(ulog.Logger())(svc), "log configuration")

	if errs := validator.Validate(svc.standardConfig); errs != nil {
		svc.Logger().Error("Invalid service configuration", "error", errs)
		return svc, errs
	}

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

	// Initialize metrics. If no metrics reporters were Registered, do noop
	// TODO(glib): add a logging reporter and use it by default, rather than noop
	// TODO: read metrics tags from config
	if svc.Metrics() == nil {
		reporter := metrics.Reporter(cfg)

		if reporter != nil {
			svc.scope = tally.NewRootScope("", nil, reporter, metrics.DefaultReporterInterval)
		} else {
			svc.scope = tally.NewRootScope("", nil, tally.NullStatsReporter, 0)
		}

		metrics.Freeze()
	}

	// if we have an observer, look for a property called "config" and load the service
	// node into that config.
	if svc.observer != nil {
		loadInstanceConfig(svc.configProvider, "service", svc.observer)

		if shc, ok := svc.observer.(SetContainerer); ok {
			shc.SetContainer(svc)
		}
	}

	// Put service into Initialized state
	svc.transitionState(Initialized)

	svc.Metrics().Counter("boot").Inc(1)

	return svc, nil
}

func ensureThat(err error, description string) {
	if err != nil {
		ulog.Logger().Fatal("Fatal error initializing app: ", description, "error", err)
	}
}
