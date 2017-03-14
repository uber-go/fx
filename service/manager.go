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

	"github.com/pkg/errors"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

const (
	defaultStartupWait = 10 * time.Second
)

// A ExitCallback is a function to handle a service shutdown and provide
// an exit code
type ExitCallback func(shutdown Exit) int

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
	m := &manager{
		// TODO: get these out of config struct instead
		moduleWrappers: []*moduleWrapper{},
		serviceCore:    serviceCore{graph: dig.New()},
	}
	m.roles = map[string]bool{}
	for _, r := range m.standardConfig.Roles {
		m.roles[r] = true
	}
	for _, opt := range builder.options {
		if optionErr := opt(m); optionErr != nil {
			return nil, errors.Wrap(optionErr, "option failed to apply")
		}
	}
	if m.configProvider == nil {
		// If the user didn't pass in a configuration provider, load the standard.
		// Bypassing standard config load is pretty much only used for tests, although it could be
		// useful in certain circumstances.
		m.configProvider = config.Load()
	}
	if err := m.setupStandardConfig(); err != nil {
		return nil, err
	}
	// Initialize metrics. If no metrics reporters were Registered, do nop
	// TODO(glib): add a logging reporter and use it by default, rather than nop
	m.setupMetrics()
	if err := m.setupLogging(); err != nil {
		return nil, err
	}
	m.setupAuthClient()
	if err := m.setupRuntimeMetricsCollector(); err != nil {
		return nil, err
	}
	m.setupVersionMetricsEmitter()
	if err := m.setupTracer(); err != nil {
		return nil, err
	}
	// if we have an observer, look for a property called "config" and load the service
	// node into that config.
	m.setupObserver()
	m.transitionState(Initialized)
	m.Metrics().Counter("boot").Inc(1)

	// register host
	var h Host = m
	m.graph.MustRegister(&h)

	for _, moduleInfo := range builder.moduleInfos {
		if err := m.addModule(moduleInfo.provider, moduleInfo.options...); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *manager) IsRunning() bool {
	return m.closeChan != nil
}

func (m *manager) OnCriticalError(err error) {
	shutdown := true
	if m.observer == nil {
		zap.L().Warn(
			"No observer set to handle lifecycle events. Shutting down.",
			zap.String("event", "OnCriticalError"),
		)
	} else {
		shutdown = !m.observer.OnCriticalError(err)
	}

	if shutdown {
		if ok, err := m.shutdown(err, "", nil); !ok || err != nil {
			// TODO(ai) verify we flush logs
			zap.L().Info("Problem shutting down module",
				zap.Bool("success", ok),
				zap.Error(err),
			)
		}
	}
}

// StartAsync service is used as a non-blocking the call on service start. StartAsync will
// return the call to the caller with a Control to listen on channels
// and trigger manual shutdowns.
func (m *manager) StartAsync() Control {
	return m.start()
}

// Start service is used for blocking the call on service start. Start will block the
// call and yield the control to the service lifecyce manager. Start will not yield back
// the control to the caller, so no code will be executed after calling Start()
func (m *manager) Start() {
	m.start()
	// block until forced exit
	m.WaitForShutdown(nil)
}

// Stop shuts down the service.
func (m *manager) Stop(reason string, exitCode int) error {
	ec := &exitCode
	_, err := m.shutdown(nil, reason, ec)
	return err
}

func (m *manager) WaitForShutdown(exitCallback ExitCallback) {
	shutdown := <-m.closeChan
	zap.L().Info("Shutting down", zap.String("reason", shutdown.Reason))

	exit := 0
	if exitCallback != nil {
		exit = exitCallback(shutdown)
	} else if shutdown.Error != nil {
		exit = 1
	}
	os.Exit(exit)
}

func (m *manager) supportsRole(roles ...string) bool {
	if len(m.roles) == 0 || len(roles) == 0 {
		return true
	}

	for _, role := range roles {
		if found, ok := m.roles[role]; found && ok {
			return true
		}
	}
	return false
}

func (m *manager) shutdown(err error, reason string, exitCode *int) (bool, error) {
	m.shutdownMu.Lock()
	m.inShutdown = true
	defer func() {
		m.inShutdown = false
		m.shutdownMu.Unlock()
	}()

	if m.shutdownReason != nil || !m.IsRunning() {
		return false, nil
	}

	m.transitionState(Stopping)

	m.shutdownReason = &Exit{
		Reason:   reason,
		Error:    err,
		ExitCode: 0,
	}

	if err != nil {
		if reason != "" {
			m.shutdownReason.Reason = err.Error()
		}
		m.shutdownReason.ExitCode = 1
	}

	if exitCode != nil {
		m.shutdownReason.ExitCode = *exitCode
	}

	// Log the module shutdown errors
	errs := m.stopModules()
	if len(errs) > 0 {
		for _, err := range errs {
			zap.L().Error("Failure to shut down module", zap.Error(err))
		}
	}

	// Stop runtime metrics collection. Uses scope, should be closed before scope is closed.
	if m.runtimeCollector != nil {
		m.runtimeCollector.Close()
	}

	if m.versionEmitter != nil {
		m.versionEmitter.close()
	}

	// Stop the metrics reporting
	if m.metricsCloser != nil {
		if err = m.metricsCloser.Close(); err != nil {
			zap.L().Error("Failure to close metrics", zap.Error(err))
		}
	}

	// Flush tracing buffers
	if m.tracerCloser != nil {
		zap.L().Debug("Closing tracer")
		if err = m.tracerCloser.Close(); err != nil {
			zap.L().Error("Failure to close tracer", zap.Error(err))
		}
	}

	// report that we shutdown.
	m.closeChan <- *m.shutdownReason
	close(m.closeChan)

	if m.observer != nil {
		m.observer.OnShutdown(*m.shutdownReason)
	}

	m.transitionState(Stopped)

	return true, err
}

func (m *manager) addModule(provider ModuleProvider, options ...ModuleOption) error {
	if m.locked {
		return fmt.Errorf("can't add module: service already started")
	}
	moduleWrapper, err := newModuleWrapper(m, provider, options...)
	if err != nil {
		return err
	}
	if moduleWrapper == nil {
		return nil
	}
	if !m.supportsRole(moduleWrapper.scopedHost.Roles()...) {
		zap.L().Info(
			"module will not be added due to selected roles",
			zap.Any("roles", moduleWrapper.scopedHost.Roles()),
		)
	}
	m.moduleWrappers = append(m.moduleWrappers, moduleWrapper)
	return nil
}

func (m *manager) start() Control {
	m.locked = true
	m.shutdownMu.Lock()
	m.transitionState(Starting)

	readyCh := make(chan struct{}, 1)
	defer func() {
		readyCh <- struct{}{}
	}()
	if m.inShutdown {
		m.shutdownMu.Unlock()
		return Control{
			ReadyChan:    readyCh,
			ServiceError: errors.New("shutting down the service"),
		}
	} else if m.IsRunning() {
		m.shutdownMu.Unlock()
		return Control{
			ExitChan:     m.closeChan,
			ReadyChan:    readyCh,
			ServiceError: errors.New("service is already running"),
		}
	} else {
		if m.observer != nil {
			if err := m.observer.OnInit(m); err != nil {
				m.shutdownMu.Unlock()
				return Control{
					ReadyChan:    readyCh,
					ServiceError: errors.Wrap(err, "failed to initialize the observer"),
				}
			}
		}
		m.shutdownReason = nil
		m.closeChan = make(chan Exit, 1)
		errs := m.startModules()
		m.registerSignalHandlers()
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

				m.shutdownMu.Unlock()
				if _, err := m.shutdown(e, "", nil); err != nil {
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

	m.transitionState(Running)
	m.shutdownMu.Unlock()

	return Control{
		ExitChan:  m.closeChan,
		ReadyChan: readyCh,
	}
}

func (m *manager) registerSignalHandlers() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		zap.L().Warn("Received shutdown signal", zap.String("signal", sig.String()))
		if err := m.Stop("Received syscall", 0); err != nil {
			zap.L().Error("Error shutting down", zap.Error(err))
		}
	}()
}

func (m *manager) startModules() []error {
	var results []error
	var lock sync.Mutex
	wg := sync.WaitGroup{}

	// make sure we wait for all the start
	// calls to return
	wg.Add(len(m.moduleWrappers))
	for _, mod := range m.moduleWrappers {
		go func(mw *moduleWrapper) {
			if !mw.IsRunning() {
				errC := make(chan error, 1)
				go func() { errC <- mw.Start() }()
				select {
				case err := <-errC:
					if err != nil {
						zap.L().Error(
							"Error received while starting module",
							zap.String("module", mw.Name()),
							zap.Error(err),
						)
						lock.Lock()
						results = append(results, err)
						lock.Unlock()
					} else {
						zap.L().Info("Module started up cleanly", zap.String("module", mw.Name()))
					}
				case <-time.After(defaultStartupWait):
					lock.Lock()
					results = append(
						results,
						fmt.Errorf("module: %s didn't start after %v", mw.Name(), defaultStartupWait),
					)
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

func (m *manager) stopModules() []error {
	var results []error
	var lock sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(m.moduleWrappers))
	for _, mod := range m.moduleWrappers {
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

func (m *manager) transitionState(to State) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	// TODO(ai) this isn't used yet
	if to < m.state {
		zap.L().Fatal("Can't down from state",
			zap.Any("from", m.state),
			zap.Any("to", to),
			zap.String("service", m.Name()),
		)
	}

	for m.state < to {
		old := m.state
		newState := m.state
		switch m.state {
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
		m.state = newState
		if m.observer != nil {
			m.observer.OnStateChange(old, newState)
		}
	}
}
