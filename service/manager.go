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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/zap"

	"github.com/pkg/errors"
)

const (
	defaultStartupWait = time.Second
)

// Implements Manager interface
type manager struct {
	serviceCore
	locked         bool
	observer       Observer
	moduleWrappers []*moduleWrapper
	roles          map[string]bool
	stateMu        sync.Mutex

	// Shutdown fields.
	shutdownMu     sync.Mutex
	inShutdown     bool  // Protected by shutdownMu
	shutdownReason *Exit // Protected by shutdownMu
	closeChan      chan Exit
	started        bool
}

// TODO(glib): manager is both an Manager and a Host?
var _ Host = &manager{}
var _ Manager = &manager{}

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

func (s *manager) addModuleWrapper(moduleWrapper *moduleWrapper) error {
	s.moduleWrappers = append(s.moduleWrappers, moduleWrapper)
	return nil
}

func (s *manager) supportsRole(roles ...string) bool {
	if len(s.roles) == 0 || len(roles) == 0 {
		return true
	}

	for _, role := range roles {
		if found, ok := s.roles[role]; found && ok {
			return true
		}
	}
	return false
}

func (s *manager) IsRunning() bool {
	return s.closeChan != nil
}

func (s *manager) OnCriticalError(err error) {
	shutdown := true
	if s.observer == nil {
		zap.L().Warn(
			"No observer set to handle lifecycle events. Shutting down.",
			zap.String("event", "OnCriticalError"),
		)
	} else {
		shutdown = !s.observer.OnCriticalError(err)
	}

	if shutdown {
		if ok, err := s.shutdown(err, "", nil); !ok || err != nil {
			// TODO(ai) verify we flush logs
			zap.L().Info("Problem shutting down module",
				zap.Bool("success", ok),
				zap.Error(err),
			)
		}
	}
}

func (s *manager) shutdown(err error, reason string, exitCode *int) (bool, error) {
	s.shutdownMu.Lock()
	s.inShutdown = true
	defer func() {
		s.inShutdown = false
		s.shutdownMu.Unlock()
	}()

	if s.shutdownReason != nil || !s.IsRunning() {
		return false, nil
	}

	s.transitionState(Stopping)

	s.shutdownReason = &Exit{
		Reason:   reason,
		Error:    err,
		ExitCode: 0,
	}

	if err != nil {
		if reason != "" {
			s.shutdownReason.Reason = err.Error()
		}
		s.shutdownReason.ExitCode = 1
	}

	if exitCode != nil {
		s.shutdownReason.ExitCode = *exitCode
	}

	// Log the module shutdown errors
	errs := s.stopModules()
	if len(errs) > 0 {
		for _, err := range errs {
			zap.L().Error("Failure to shut down module", zap.Error(err))
		}
	}

	// Stop runtime metrics collection. Uses scope, should be closed before scope is closed.
	if s.runtimeCollector != nil {
		s.runtimeCollector.Close()
	}

	// Stop the metrics reporting
	if s.metricsCloser != nil {
		if err = s.metricsCloser.Close(); err != nil {
			zap.L().Error("Failure to close metrics", zap.Error(err))
		}
	}

	// Flush tracing buffers
	if s.tracerCloser != nil {
		zap.L().Debug("Closing tracer")
		if err = s.tracerCloser.Close(); err != nil {
			zap.L().Error("Failure to close tracer", zap.Error(err))
		}
	}

	// report that we shutdown.
	s.closeChan <- *s.shutdownReason
	close(s.closeChan)

	if s.observer != nil {
		s.observer.OnShutdown(*s.shutdownReason)
	}

	s.transitionState(Stopped)

	return true, err
}

func (s *manager) addModule(provider ModuleProvider, options ...ModuleOption) error {
	if s.locked {
		return fmt.Errorf("can't add module: service already started")
	}
	moduleWrapper, err := newModuleWrapper(s, provider, options...)
	if err != nil {
		return err
	}
	if moduleWrapper == nil {
		return nil
	}
	if !s.supportsRole(moduleWrapper.scopedHost.Roles()...) {
		zap.L().Info(
			"module will not be added due to selected roles",
			zap.Any("roles", moduleWrapper.scopedHost.Roles()),
		)
	}
	return s.addModuleWrapper(moduleWrapper)
}

// StartAsync service is used as a non-blocking the call on service start. StartAsync will
// return the call to the caller with a Control to listen on channels
// and trigger manual shutdowns.
func (s *manager) StartAsync() Control {
	return s.start()
}

// Start service is used for blocking the call on service start. Start will block the
// call and yield the control to the service lifecyce manager. Start will not yield back
// the control to the caller, so no code will be executed after calling Start()
func (s *manager) Start() {
	s.start()

	// block until forced exit
	s.WaitForShutdown(nil)
}

func (s *manager) start() Control {
	var err error
	s.locked = true
	s.shutdownMu.Lock()
	s.transitionState(Starting)

	readyCh := make(chan struct{}, 1)
	defer func() {
		readyCh <- struct{}{}
	}()
	if s.inShutdown {
		s.shutdownMu.Unlock()
		return Control{
			ReadyChan:    readyCh,
			ServiceError: errors.New("shutting down the service"),
		}
	} else if s.IsRunning() {
		s.shutdownMu.Unlock()
		return Control{
			ExitChan:     s.closeChan,
			ReadyChan:    readyCh,
			ServiceError: errors.New("service is already running"),
		}
	} else {
		if s.observer != nil {
			if err := s.observer.OnInit(s); err != nil {
				s.shutdownMu.Unlock()
				return Control{
					ReadyChan:    readyCh,
					ServiceError: errors.Wrap(err, "failed to initialize the observer"),
				}
			}
		}
		s.shutdownReason = nil
		s.closeChan = make(chan Exit, 1)
		errs := s.startModules()
		s.registerSignalHandlers()
		if len(errs) > 0 {
			var serviceErr error
			errChan := make(chan Exit, 1)
			// grab the first error, shut down the service and return the error
			for _, e := range errs {
				errChan <- Exit{
					Error:    e,
					Reason:   "Module start failed",
					ExitCode: 4,
				}

				s.shutdownMu.Unlock()
				if _, err := s.shutdown(e, "", nil); err != nil {
					zap.L().Error("Unable to shut down modules",
						zap.NamedError("initialError", e),
						zap.NamedError("shutdownError", err),
					)
				}
				zap.L().Error("Error starting the module", zap.Error(e))
				// return first service error
				if serviceErr == nil {
					serviceErr = e
				}
			}
			return Control{
				ExitChan:     errChan,
				ReadyChan:    readyCh,
				ServiceError: serviceErr,
			}
		}
	}

	s.transitionState(Running)
	s.shutdownMu.Unlock()

	return Control{
		ExitChan:     s.closeChan,
		ReadyChan:    readyCh,
		ServiceError: err,
	}
}

func (s *manager) registerSignalHandlers() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		zap.L().Warn("Received shutdown signal", zap.String("signal", sig.String()))
		if err := s.Stop("Received syscall", 0); err != nil {
			zap.L().Error("Error shutting down", zap.Error(err))
		}
	}()
}

// Stop shuts down the service.
func (s *manager) Stop(reason string, exitCode int) error {
	ec := &exitCode
	_, err := s.shutdown(nil, reason, ec)
	return err
}

func (s *manager) startModules() []error {
	var results []error
	var lock sync.Mutex
	wg := sync.WaitGroup{}

	// make sure we wait for all the start
	// calls to return
	wg.Add(len(s.moduleWrappers))
	for _, mod := range s.moduleWrappers {
		go func(m *moduleWrapper) {
			if !m.IsRunning() {
				errC := make(chan error, 1)
				go func() { errC <- m.Start() }()
				select {
				case err := <-errC:
					if err != nil {
						zap.L().Error("Error received while starting module", zap.String("module", m.Name()), zap.Error(err))
						lock.Lock()
						results = append(results, err)
						lock.Unlock()
					} else {
						zap.L().Info("Module started up cleanly", zap.String("module", m.Name()))
					}
				case <-time.After(defaultStartupWait):
					lock.Lock()
					results = append(results, fmt.Errorf("module didn't start after %v", defaultStartupWait))
					lock.Unlock()
				}
			}
			wg.Done()
		}(mod)
	}

	// wait for the modules to all start
	wg.Wait()
	return results
}

func (s *manager) stopModules() []error {
	var results []error
	var lock sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(s.moduleWrappers))
	for _, mod := range s.moduleWrappers {
		go func(m *moduleWrapper) {
			if !m.IsRunning() {
				// TODO: have a timeout here so a bad shutdown
				// doesn't block everyone
				if err := m.Stop(); err != nil {
					lock.Lock()
					results = append(results, err)
					lock.Unlock()
				}
			}
			wg.Done()
		}(mod)
	}
	wg.Wait()
	return results
}

// A ExitCallback is a function to handle a service shutdown and provide
// an exit code
type ExitCallback func(shutdown Exit) int

func (s *manager) WaitForShutdown(exitCallback ExitCallback) {
	shutdown := <-s.closeChan
	zap.L().Info("Shutting down", zap.String("reason", shutdown.Reason))

	exit := 0
	if exitCallback != nil {
		exit = exitCallback(shutdown)
	} else if shutdown.Error != nil {
		exit = 1
	}
	os.Exit(exit)
}

func (s *manager) transitionState(to State) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	// TODO(ai) this isn't used yet
	if to < s.state {
		zap.L().Fatal("Can't down from state",
			zap.Any("from", s.state),
			zap.Any("to", to),
			zap.String("service", s.Name()),
		)
	}

	for s.state < to {
		old := s.state
		newState := s.state
		switch s.state {
		case Uninitialized:
			newState = Initialized
		case Initialized:
			newState = Starting
		case Starting:
			newState = Running
		case Running:
			newState = Stopping
		case Stopping:
			newState = Stopped
		case Stopped:
		}
		s.state = newState
		if s.observer != nil {
			s.observer.OnStateChange(old, newState)
		}
	}
}
