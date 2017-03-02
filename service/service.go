// Copyright (c) 2017 Uber Technologies, Inc.
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
	"go.uber.org/fx/config"

	"github.com/pkg/errors"
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

// A Manager encapsulates service ownership
type Manager interface {
	Host

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
	Name        string   `yaml:"name" validate:"nonzero"`
	Owner       string   `yaml:"owner"  validate:"nonzero"`
	Description string   `yaml:"description"`
	Roles       []string `yaml:"roles"`
}

// newManager creates a service Manager from a Builder.
func newManager(builder *Builder) (Manager, error) {
	svc := &manager{
		// TODO: get these out of config struct instead
		moduleWrappers: []*moduleWrapper{},
		serviceCore:    serviceCore{},
	}

	// hash up the roles
	svc.roles = map[string]bool{}
	for _, r := range svc.standardConfig.Roles {
		svc.roles[r] = true
	}

	// Run the rest of the options
	for _, opt := range builder.options {
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
	if err := svc.setupStandardConfig(); err != nil {
		return nil, err
	}

	// Initialize metrics. If no metrics reporters were Registered, do nop
	// TODO(glib): add a logging reporter and use it by default, rather than nop
	svc.setupMetrics()

	if err := svc.setupLogging(); err != nil {
		return nil, err
	}

	svc.setupAuthClient()

	if err := svc.setupRuntimeMetricsCollector(); err != nil {
		return nil, err
	}

	if err := svc.setupTracer(); err != nil {
		return nil, err
	}

	// if we have an observer, look for a property called "config" and load the service
	// node into that config.
	svc.setupObserver()

	// Put service into Initialized state
	svc.transitionState(Initialized)

	svc.Metrics().Counter("boot").Inc(1)

	for _, moduleInfo := range builder.moduleInfos {
		if err := svc.addModule(moduleInfo.provider, moduleInfo.options...); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

type niladicStart func()

func (n niladicStart) OnInit(service Host) error      { return nil }
func (n niladicStart) OnShutdown(reason Exit)         {}
func (n niladicStart) OnCriticalError(err error) bool { return true }
func (n niladicStart) OnStateChange(old State, curr State) {
	if old == Starting && curr == Running {
		n()
	}
}

// AfterStart will create an observer that will execute f() immediately after service starts.
func AfterStart(f func()) Observer {
	return niladicStart(f)
}
