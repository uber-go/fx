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
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/go-validator/validator"
	"github.com/pkg/errors"
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

	// Start service is used for blocking the call on service start. Start will block the
	// call and yield the control to the service lifecyce manager. No code will be executed
	//after call to Start() the service.
	Start()

	// StartAsync service is used as a non-blocking the call on service start. StartAsync will
	// return the call to the caller with a Control to listen on channels
	// and trigger manual shutdown.
	StartAsync() Control
	Stop(reason string, exitCode int) error
}

// Control keeps the listening channels from the service startup
type Control struct {
	ExitChan     chan Exit
	ReadyChan    chan struct{}
	ServiceError error
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
			resources: map[string]interface{}{},
		},
	}

	// hash up the roles
	svc.roles = map[string]bool{}
	for _, r := range svc.standardConfig.ServiceRoles {
		svc.roles[r] = true
	}

	// Run the rest of the options
	for _, opt := range options {
		if optionErr := opt(svc); optionErr != nil {
			return nil, errors.Wrap(optionErr, "option failed to apply")
		}
	}

	if svc.configProvider == nil {
		// If the user didn't pass in a configuration provider, load the standard.
		// Bypassing standard config load is pretty much only used for tests, although it could be
		// useful in certain circumstances.
		svc.configProvider = config.Load()
	}

	// load standard config
	// TODO(glib): `.Get("")` is a confusing interface for getting the root config node
	if err := svc.configProvider.Get("").PopulateStruct(&svc.standardConfig); err != nil {
		return nil, errors.Wrap(err, "unable to load standard configuration")
	}

	if svc.log == nil {
		logBuilder := ulog.Builder()
		// load and configure logging
		err := svc.configProvider.Get("logging").PopulateStruct(&svc.logConfig)
		if err != nil {
			ulog.Logger().Info("Logging configuration is not provided, setting to default logger", "error", err)
		}
		svc.log = logBuilder.WithConfiguration(svc.logConfig).Build()
	} else {
		svc.log.Debug("Using custom log provider due to service.WithLogger option")
	}

	if errs := validator.Validate(svc.standardConfig); errs != nil {
		svc.Logger().Error("Invalid service configuration", "error", errs)
		return svc, errors.Wrap(errs, "service configuration failed validation")
	}

	// Initialize metrics. If no metrics reporters were Registered, do noop
	// TODO(glib): add a logging reporter and use it by default, rather than noop
	if svc.Metrics() == nil {
		svc.scope = metrics.RootScope(svc)

		if svc.scope == nil {
			svc.scope = tally.NoopScope
		}

		metrics.Freeze()
	}

	if svc.RuntimeMetricsCollector() == nil {
		var runtimeMetricsConfig metrics.RuntimeConfig
		err := svc.configProvider.Get("runtimeMetrics").PopulateStruct(&runtimeMetricsConfig)
		if err != nil {
			return nil, errors.Wrap(err, "unable to load runtime metrics configuration")
		}
		svc.runtimeCollector = metrics.StartCollectingRuntimeMetrics(
			svc.scope.SubScope("runtime"), time.Second, runtimeMetricsConfig,
		)
	}

	if svc.Tracer() == nil {
		if err := svc.configProvider.Get("tracing").PopulateStruct(&svc.tracerConfig); err != nil {
			return nil, errors.Wrap(err, "unable to load tracing configuration")
		}
		tracer, closer, err := tracing.InitGlobalTracer(
			&svc.tracerConfig,
			svc.standardConfig.ServiceName,
			svc.log,
			svc.scope.SubScope("tracing"),
		)
		if err != nil {
			return svc, errors.Wrap(err, "unable to initialize global tracer")
		}
		svc.tracer = tracer
		svc.tracerCloser = closer
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
