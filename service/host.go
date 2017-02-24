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
	"reflect"
	"sync"
	"syscall"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/internal/util"
	"go.uber.org/zap"

	"github.com/pkg/errors"
)

const (
	defaultStartupWait = time.Second
)

// Implements Manager interface
type manager struct {
	serviceCore
	locked   bool
	observer Observer
	modules  []Module
	roles    map[string]bool
	stateMu  sync.Mutex

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

func (s *manager) addModule(module Module) error {
	if s.locked {
		return fmt.Errorf("can't add module: service already started")
	}
	s.modules = append(s.modules, module)
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

func (s *manager) Modules() []Module {
	mods := make([]Module, len(s.modules))
	copy(mods, s.modules)
	return mods
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
		for k, v := range errs {
			zap.L().Error("Failure to shut down module",
				zap.String("name", k.Name()),
				zap.Error(v),
			)
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

// addModules adds the given modules to a service manager
func (s *manager) addModules(modules ...ModuleCreateFunc) error {
	for _, mcf := range modules {
		mi := ModuleCreateInfo{
			Host:  s,
			Items: make(map[string]interface{}),
		}

		mods, err := mcf(mi)
		if err != nil {
			return err
		}

		if !s.supportsRole(mi.Roles...) {
			zap.L().Info(
				"module will not be added due to selected roles",
				zap.Any("roles", mi.Roles),
			)
		}

		for _, mod := range mods {
			err = s.addModule(mod)
		}
	}

	return nil
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

func (s *manager) startModules() map[Module]error {
	results := map[Module]error{}
	wg := sync.WaitGroup{}

	// make sure we wait for all the start
	// calls to return
	wg.Add(len(s.modules))
	for _, mod := range s.modules {
		go func(m Module) {
			if !m.IsRunning() {
				readyCh := make(chan struct{}, 1)
				startResult := m.Start(readyCh)

				select {
				case <-readyCh:
					zap.L().Info("Module started up cleanly", zap.String("module", m.Name()))
				case <-time.After(defaultStartupWait):
					results[m] = fmt.Errorf("module didn't start after %v", defaultStartupWait)
				}

				if startError := <-startResult; startError != nil {
					zap.L().Error("Error received while starting module", zap.String("module", m.Name()), zap.Error(startError))
					results[m] = startError
				}
			}
			wg.Done()
		}(mod)
	}

	// wait for the modules to all start
	wg.Wait()
	return results
}

func (s *manager) stopModules() map[Module]error {
	results := map[Module]error{}
	wg := sync.WaitGroup{}
	wg.Add(len(s.modules))
	for _, mod := range s.modules {
		go func(m Module) {
			if !m.IsRunning() {
				// TODO: have a timeout here so a bad shutdown
				// doesn't block everyone
				if err := m.Stop(); err != nil {
					results[m] = err
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

const instanceConfigName = "ServiceConfig"

func loadInstanceConfig(cfg config.Provider, key string, instance interface{}) bool {
	fieldName := instanceConfigName
	if field, found := util.FindField(instance, &fieldName, nil); found {

		configValue := reflect.New(field.Type())

		// Try to load the service config
		err := cfg.Get(key).PopulateStruct(configValue.Interface())
		if err != nil {
			zap.L().Error("Unable to load instance config", zap.Error(err))
			return false
		}
		instanceValue := reflect.ValueOf(instance).Elem()
		instanceValue.FieldByName(fieldName).Set(configValue.Elem())
		return true
	}
	return false
}
