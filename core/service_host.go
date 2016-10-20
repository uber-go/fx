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
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"time"

	"go.uber.org/fx/core/config"
	"go.uber.org/fx/core/ulog"
	"go.uber.org/fx/internal/util"
)

const (
	defaultStartupWait = time.Second
)

type serviceHost struct {
	serviceCore
	locked   bool
	observer Observer
	modules  []Module
	roles    map[string]bool

	// Shutdown fields.
	shutdownMu     sync.Mutex
	inShutdown     bool         // Protected by shutdownMu
	shutdownReason *ServiceExit // Protected by shutdownMu
	closeChan      chan ServiceExit
	started        bool
}

var _ ServiceHost = &serviceHost{}
var _ ServiceOwner = &serviceHost{}

func (s *serviceHost) addModule(module Module) error {
	if s.locked {
		return fmt.Errorf("can't add module: service already started")
	}
	s.modules = append(s.modules, module)
	return module.Initialize(s)
}

func (s *serviceHost) supportsRole(roles ...string) bool {
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

func (s *serviceHost) Modules() []Module {
	mods := make([]Module, len(s.modules))
	copy(mods, s.modules)
	return mods
}

func (s *serviceHost) IsRunning() bool {
	return s.closeChan != nil
}

func (s *serviceHost) OnCriticalError(err error) {
	shutdown := true
	if s.observer == nil {
		s.Logger().With("event", "OnCriticalError").Warn("No observer set to handle lifecycle events. Shutting down.")
	} else {
		shutdown = !s.observer.OnCriticalError(err)
	}

	if shutdown {
		if ok, err := s.shutdown(err, "", nil); !ok || err != nil {
			// TODO(ai) verify we flush logs
			s.Logger().With("success", ok, "error", err).Info("Problem shutting down module")
		}
	}
}

func (s *serviceHost) shutdown(err error, reason string, exitCode *int) (bool, error) {
	s.shutdownMu.Lock()
	s.inShutdown = true
	defer func() {
		s.inShutdown = false
		s.shutdownMu.Unlock()
	}()

	if s.shutdownReason != nil || !s.IsRunning() {
		return false, nil
	}

	s.shutdownReason = &ServiceExit{
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

	s.stopModules()

	// TODO: What do we do with shutdown errors?
	// if len(errs) > 0 {
	// 	errList := "errModuleStopError\n"
	// 	for k, v := range errs {
	// 		errList += fmt.Sprintf("Module %q: %s\n", k.Name(), v.Error())
	// 	}

	// }

	// report that we shutdown.
	s.closeChan <- *s.shutdownReason
	close(s.closeChan)

	if s.observer != nil {
		s.observer.OnShutdown(*s.shutdownReason)
	}
	return true, err
}

// Start runs the serve loop. If Shutdown() was called then the shutdown reason
// will be returned.
func (s *serviceHost) Start(waitForShutdown bool) (<-chan ServiceExit, <-chan struct{}, error) {
	var err error
	s.locked = true
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

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
				return nil, readyCh, err
			}
		}
		s.shutdownReason = nil
		s.closeChan = make(chan ServiceExit)
		errs := s.startModules()
		if len(errs) > 0 {
			// grab the first error, shut down the service
			// and return the error
			for _, e := range errs {

				errChan := make(chan ServiceExit)
				errChan <- *s.shutdownReason
				if _, err := s.shutdown(e, "", nil); err != nil {
					ulog.Logger().With("initialError", e, "shutdownError", err).Error("Unable to shut down modules")
				}

				return errChan, readyCh, e
			}
		}
	}

	if waitForShutdown {
		s.WaitForShutdown(nil)
	}

	return s.closeChan, readyCh, err
}

// Stop shuts down the service.
func (s *serviceHost) Stop(reason string, exitCode int) error {
	ec := &exitCode
	_, err := s.shutdown(nil, reason, ec)
	return err
}

func (s *serviceHost) startModules() map[Module]error {
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
				if startError := <-startResult; startError != nil {
					results[m] = startError
				}
				select {
				case <-readyCh:
					s.Logger().With("module", m.Name()).Debug("Module started up cleanly")
				case <-time.After(defaultStartupWait):
					results[m] = fmt.Errorf("module didn't start after %v", defaultStartupWait)
				}
			}
			wg.Done()
		}(mod)
	}

	// wait for the modules to all start
	wg.Wait()
	return results
}

func (s *serviceHost) stopModules() map[Module]error {
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

// A ServiceExitCallback is a function to handle a service shutdown and provide
// an exit code
type ServiceExitCallback func(shutdown ServiceExit) int

func (s *serviceHost) WaitForShutdown(exitCallback ServiceExitCallback) {
	shutdown := <-s.closeChan
	log.Printf("\nShutting down because %q\n", shutdown.Reason)

	exit := 0
	if exitCallback != nil {
		exit = exitCallback(shutdown)
	} else if shutdown.Error != nil {
		exit = 1
	}
	os.Exit(exit)
}

func (s *serviceHost) transitionState(to ServiceState) {
	// TODO(ai) this isn't used yet
	if to < s.state {
		s.Logger().With("service", s.Name()).Fatal("Can't down from state", "from", s.state, "to", to)
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

func loadInstanceConfig(cfg config.ConfigurationProvider, key string, instance interface{}) bool {
	fieldName := instanceConfigName
	if field, found := util.FindField(instance, &fieldName, nil); found {

		configValue := reflect.New(field.Type())

		// Try to load the service config
		if cfg.GetValue(key).PopulateStruct(configValue.Interface()) {
			instanceValue := reflect.ValueOf(instance).Elem()
			instanceValue.FieldByName(fieldName).Set(configValue.Elem())
			return true
		}
	}
	return false
}
