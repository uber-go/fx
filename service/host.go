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
	gcontext "context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"go.uber.org/fx/core/config"
	"go.uber.org/fx/core/ulog"
	"go.uber.org/fx/internal/util"
)

const (
	defaultStartupWait = time.Second
)

type host struct {
	serviceCore
	locked   bool
	observer Observer
	modules  []Module
	roles    map[string]bool

	// Shutdown fields.
	shutdownMu     sync.Mutex
	inShutdown     bool  // Protected by shutdownMu
	shutdownReason *Exit // Protected by shutdownMu
	closeChan      chan Exit
	started        bool
}

// TODO(glib): host is both an Owner and a Host?
var _ Host = &host{}
var _ Owner = &host{}

func (s *host) addModule(module Module) error {
	if s.locked {
		return fmt.Errorf("can't add module: service already started")
	}
	s.modules = append(s.modules, module)
	return module.Initialize(NewContext(gcontext.Background(), s))
}

func (s *host) supportsRole(roles ...string) bool {
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

func (s *host) Modules() []Module {
	mods := make([]Module, len(s.modules))
	copy(mods, s.modules)
	return mods
}

func (s *host) IsRunning() bool {
	return s.closeChan != nil
}

func (s *host) OnCriticalError(err error) {
	shutdown := true
	if s.observer == nil {
		s.Logger().Warn(
			"No observer set to handle lifecycle events. Shutting down.",
			"event", "OnCriticalError",
		)
	} else {
		shutdown = !s.observer.OnCriticalError(err)
	}

	if shutdown {
		if ok, err := s.shutdown(err, "", nil); !ok || err != nil {
			// TODO(ai) verify we flush logs
			s.Logger().Info("Problem shutting down module", "success", ok, "error", err)
		}
	}
}

func (s *host) shutdown(err error, reason string, exitCode *int) (bool, error) {
	s.shutdownMu.Lock()
	s.inShutdown = true
	defer func() {
		s.inShutdown = false
		s.shutdownMu.Unlock()
	}()

	if s.shutdownReason != nil || !s.IsRunning() {
		return false, nil
	}

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
			s.Logger().Error("Failure to shut down module", "name", k.Name(), "error", v.Error())
		}
	}

	// Stop the metrics reporting
	if s.scope != nil {
		s.scope.Close()
	}

	// Flush tracing buffers
	if s.tracerCloser != nil {
		s.Logger().Debug("Closing tracer")
		if err = s.tracerCloser.Close(); err != nil {
			s.Logger().Error("Failure to close tracer", "error", err)
		}
	}

	// report that we shutdown.
	s.closeChan <- *s.shutdownReason
	close(s.closeChan)

	if s.observer != nil {
		s.observer.OnShutdown(*s.shutdownReason)
	}
	return true, err
}

// AddModules adds the given modules to a service host
func (s *host) AddModules(modules ...ModuleCreateFunc) error {
	for _, mcf := range modules {
		mi := ModuleCreateInfo{
			Items: make(map[string]interface{}),
			Ctx:   NewContext(gcontext.Background(), s),
		}

		mods, err := mcf(mi)
		if err != nil {
			return err
		}

		if !s.supportsRole(mi.Roles...) {
			s.Logger().Info(
				"module will not be added due to selected roles",
				"roles", mi.Roles,
			)
		}

		for _, mod := range mods {
			err = s.addModule(mod)
		}
	}

	return nil
}

// Start runs the serve loop. If Shutdown() was called then the shutdown reason
// will be returned.
func (s *host) Start(waitForShutdown bool) (<-chan Exit, <-chan struct{}, error) {
	var err error
	s.locked = true
	s.shutdownMu.Lock()

	readyCh := make(chan struct{}, 1)
	defer func() {
		readyCh <- struct{}{}
	}()
	if s.inShutdown {
		return nil, readyCh, fmt.Errorf("errShuttingDown")
	} else if s.IsRunning() {
		return s.closeChan, readyCh, nil
	} else {
		if s.observer != nil {
			if err := s.observer.OnInit(s); err != nil {
				s.shutdownMu.Unlock()
				return nil, readyCh, err
			}
		}
		s.shutdownReason = nil
		s.closeChan = make(chan Exit, 1)
		errs := s.startModules()
		if len(errs) > 0 {
			// grab the first error, shut down the service
			// and return the error
			for _, e := range errs {
				errChan := make(chan Exit, 1)
				errChan <- Exit{
					Error:    e,
					Reason:   "Module start failed",
					ExitCode: 4,
				}

				s.shutdownMu.Unlock()
				if _, err := s.shutdown(e, "", nil); err != nil {
					s.Logger().Error("Unable to shut down modules", "initialError", e, "shutdownError", err)
				}
				return errChan, readyCh, e
			}
		}
	}

	s.shutdownMu.Unlock()
	s.registerSignalHandlers()

	if waitForShutdown {
		s.WaitForShutdown(nil)
	}

	return s.closeChan, readyCh, err
}

func (s *host) registerSignalHandlers() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		s.Logger().Warn("Received shutdown signal", "signal", sig.String())
		if err := s.Stop("Received syscall", 0); err != nil {
			s.Logger().Error("Error shutting down", "error", err.Error())
		}
	}()
}

// Stop shuts down the service.
func (s *host) Stop(reason string, exitCode int) error {
	ec := &exitCode
	_, err := s.shutdown(nil, reason, ec)
	return err
}

func (s *host) startModules() map[Module]error {
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
					s.Logger().Info("Module started up cleanly", "module", m.Name())
				case <-time.After(defaultStartupWait):
					results[m] = fmt.Errorf("module didn't start after %v", defaultStartupWait)
				}

				if startError := <-startResult; startError != nil {
					s.Logger().Error("Error received while starting module", "module", m.Name(), "error", startError)
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

func (s *host) stopModules() map[Module]error {
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

func (s *host) WaitForShutdown(exitCallback ExitCallback) {
	shutdown := <-s.closeChan
	s.Logger().Info("Shutting down", "reason", shutdown.Reason)

	exit := 0
	if exitCallback != nil {
		exit = exitCallback(shutdown)
	} else if shutdown.Error != nil {
		exit = 1
	}
	os.Exit(exit)
}

func (s *host) transitionState(to State) {
	// TODO(ai) this isn't used yet
	if to < s.state {
		s.Logger().Fatal("Can't down from state", "from", s.state, "to", to, "service", s.Name())
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
		err := cfg.GetValue(key).PopulateStruct(configValue.Interface())
		if err != nil {
			ulog.Logger().Error("Unable to load instance config", "error", err)
			return false
		}
		instanceValue := reflect.ValueOf(instance).Elem()
		instanceValue.FieldByName(fieldName).Set(configValue.Elem())
		return true
	}
	return false
}
