package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/uber-go/uberfx/core/config"
)

type Service struct {
	name    string
	desc    string
	owner   string
	modules []Module
	roles   []string

	// Shutdown fields.
	shutdownMu     sync.Mutex
	inShutdown     bool         // Protected by shutdownMu
	shutdownReason *ServiceExit // Protected by shutdownMu
	closeChan      chan ServiceExit
	started        bool
	config         serviceConfig
}

type ServiceExit struct {
	Reason   string
	Error    error
	ExitCode int
}

type serviceConfig struct {
	ServiceName        string   `yaml: "applicationid" required:"true"`
	ServiceOwner       string   `yaml: "applicationowner"  required:"true"`
	ServiceDescription string   `yaml: "applicationdesc"`
	ServiceRoles       []string `yaml: "roles"`
}

// type LoadModuleServicesFunc func(*Service) []Module

type ModuleCreateFunc func(svc *Service) ([]Module, error)

func NewService(cfg config.ConfigurationProvider, moduleCreators ...ModuleCreateFunc) *Service {

	if cfg == nil {
		cfg = config.Global()
	}

	svc := &Service{
		// TODO: get these out of config struct instead
		name:    cfg.MustGetValue(config.ApplicationIDKey).AsString(),
		desc:    cfg.MustGetValue(config.ApplicationDescriptionKey).AsString(),
		owner:   cfg.MustGetValue(config.ApplicationOwnerKey).AsString(),
		modules: []Module{},
	}
	// load config
	cfg.GetValue("", &svc.config).PopulateStruct(&svc.config)

	modulesToAdd := []Module{}

	for _, mc := range moduleCreators {
		if mod, err := mc(svc); err != nil {
			panic(err)
		} else {
			modulesToAdd = append(modulesToAdd, mod...)
		}
	}

	// make sure they all have unique names
	nameLookup := map[string]bool{}

	// hash up the roles
	rolesHash := map[string]bool{}
	for _, r := range svc.config.ServiceRoles {
		rolesHash[r] = true
	}

	for _, v := range modulesToAdd {
		if _, ok := nameLookup[v.Name()]; ok {
			panic(fmt.Sprintf("Duplicate modules of name '%s'", v.Name()))
		}
		nameLookup[v.Name()] = true

		moduleRoles := v.Roles()
		shouldAdd := true
		if len(rolesHash) > 0 && len(moduleRoles) > 0 {
			shouldAdd = false
			// make sure this modules roles intersect with the roles specified by the service
			//
			for _, r := range moduleRoles {
				if _, ok := rolesHash[r]; ok {
					shouldAdd = true
					break
				}
			}
		}
		if shouldAdd {
			svc.modules = append(svc.modules, v)
		}
	}

	return svc
}

func (s Service) Name() string {
	return s.name
}

func (s Service) Description() string {
	return s.desc
}

func (s Service) Owner() string {
	return s.owner
}
func (s Service) Roles() []string {
	return s.roles
}

func (s Service) Modules() []Module {
	mods := make([]Module, len(s.modules))
	copy(mods, s.modules)
	return mods
}

func (s *Service) isRunning() bool {
	return s.closeChan != nil
}

func (s *Service) OnCriticalError(err error) {
	s.shutdown(err, "", nil)
}

func (s *Service) shutdown(err error, reason string, exitCode *int) (bool, error) {

	s.shutdownMu.Lock()
	s.inShutdown = true
	defer func() {
		s.inShutdown = false
		s.shutdownMu.Unlock()
	}()

	if s.shutdownReason != nil || !s.isRunning() {
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

	return true, nil
}

// Start runs the serve loop. If Shutdown() was called then the shutdown reason
// will be returned.
func (s *Service) Start(waitForShutdown bool) (<-chan ServiceExit, error) {
	var err error
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	if s.inShutdown {
		return nil, errors.New("errShuttingDown")
	} else if s.isRunning() {
		return s.closeChan, nil
	} else {

		s.shutdownReason = nil
		s.closeChan = make(chan ServiceExit)
		errs := s.startModules()
		if len(errs) > 0 {
			// grab the first error, shut down the service
			// and return the error
			for _, e := range errs {

				errChan := make(chan ServiceExit)
				errChan <- *s.shutdownReason
				s.shutdown(e, "", nil)
				return errChan, e
			}
		}
	}

	if waitForShutdown {
		s.WaitForShutdown(nil)
	}

	return s.closeChan, err
}

// Stop shuts down the service.
func (s *Service) Stop(reason string, exitCode int) error {
	ec := &exitCode
	_, err := s.shutdown(nil, reason, ec)
	return err
}

func (s *Service) startModules() map[Module]error {

	results := map[Module]error{}
	wg := sync.WaitGroup{}

	// make sure we wait for all the start
	// calls to return
	wg.Add(len(s.modules))
	for _, mod := range s.modules {
		go func(m Module) {
			if !m.IsRunning() {
				startResult := m.Start()
				if startError := <-startResult; startError != nil {
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

func (s *Service) stopModules() map[Module]error {
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

type ServiceExitCallback func(shutdown ServiceExit) int

func (s *Service) WaitForShutdown(exitCallback ServiceExitCallback) {

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
