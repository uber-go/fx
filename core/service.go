package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"

	"github.com/uber-go/uberfx/core/config"
	cm "github.com/uber-go/uberfx/core/metrics"
)

type Service struct {
	locked         bool
	standardConfig serviceConfig
	modules        []Module
	roles          []string
	instance       ServiceInstance
	scope          metrics.Scope
	configProvider config.ConfigurationProvider

	// Shutdown fields.
	shutdownMu     sync.Mutex
	inShutdown     bool         // Protected by shutdownMu
	shutdownReason *ServiceExit // Protected by shutdownMu
	closeChan      chan ServiceExit
	started        bool
}

// ServiceInstance is the interface that is implemented by user service/
// code.
type ServiceInstance interface {

	// OnInit will be called after the service has been initialized
	OnInit(service *Service) error

	// OnShutdown is called before the service shuts down
	OnShutdown(reason ServiceExit)

	// OnCriticalError is called in response to a critical error.  If false
	// is returned the service will shut down, otherwise the error will be ignored.
	OnCriticalError(err error) bool
}

type ServiceExit struct {
	Reason   string
	Error    error
	ExitCode int
}

type serviceConfig struct {
	ServiceName        string   `yaml:"applicationid" required:"true"`
	ServiceOwner       string   `yaml:"applicationowner"  required:"true"`
	ServiceDescription string   `yaml:"applicationdesc"`
	ServiceRoles       []string `yaml:"roles"`
}

// type LoadModuleServicesFunc func(*Service) []Module

type ModuleCreateFunc func(svc *Service) ([]Module, error)

func New(instance ServiceInstance, options ...ServiceOption) *Service {

	cfg := config.Global()

	svc := &Service{
		// TODO: get these out of config struct instead
		modules:        []Module{},
		configProvider: cfg,
	}

	// prepend the default options.
	// TODO: This isn't right.  Do we order the options so we make sure to use
	// the passed in options?
	//
	WithConfiguration(cfg)(svc)

	// load standard config
	svc.configProvider.GetValue("", nil).PopulateStruct(&svc.standardConfig)

	WithMetricsScope(cm.Global(true))(svc)

	// Run the rest of the options
	for _, opt := range options {
		if optionErr := opt(svc); optionErr != nil {
			panic(optionErr)
		}
	}

	// make sure they all have unique names
	nameLookup := map[string]bool{}

	// hash up the roles
	rolesHash := map[string]bool{}
	for _, r := range svc.standardConfig.ServiceRoles {
		rolesHash[r] = true
	}

	for i := len(svc.modules) - 1; i >= 0; i-- {
		v := svc.modules[i]
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
		if !shouldAdd {
			// remove from the list
			svc.modules = append(svc.modules[0:i], svc.modules[i+1])
		}
	}

	// if we have an instance, look for a property called "config" and load the service
	// node into that config.
	if instance != nil {

		loadInstanceConfig(svc.configProvider, "service", instance)
		setupServiceInstance(instance, svc)
		svc.instance = instance
	}

	svc.Scope().Counter("boot").Inc(1)
	return svc
}

func loadInstanceConfig(cfg config.ConfigurationProvider, key string, instance interface{}) bool {
	instanceType := reflect.TypeOf(instance)

	// get the actual value
	for instanceType.Kind() == reflect.Ptr {
		instanceType = instanceType.Elem()
	}

	if configField, found := instanceType.FieldByName("Config"); found {
		configValue := reflect.New(configField.Type)

		// Try to load the service config
		if cfg.GetValue(key, nil).PopulateStruct(configValue.Interface()) {
			instanceValue := reflect.ValueOf(instance).Elem()
			instanceValue.FieldByName("Config").Set(configValue.Elem())
			return true
		}
	}
	return false
}

func setupServiceInstance(instance ServiceInstance, service *Service) {
	// walk the fields looking for ones of type Service
	//
	val := reflect.ValueOf(instance).Elem()
	serviceType := reflect.TypeOf(Service{})
	for i := 0; i < val.NumField(); i++ {
		field := val.FieldByIndex([]int{i})

		// is it service type field?
		if serviceType.AssignableTo(field.Type()) {
			field.Set(reflect.ValueOf(*service))
			return
		}
	}
}

func NewService(instance ServiceInstance, cfg config.ConfigurationProvider, moduleCreators ...ModuleCreateFunc) *Service {

	if cfg == nil {
		cfg = config.Global()
	}

	svc := &Service{
		// TODO: get these out of config struct instead
		// name:           cfg.MustGetValue(config.ApplicationIDKey).AsString(),
		// desc:           cfg.MustGetValue(config.ApplicationDescriptionKey).AsString(),
		// owner:          cfg.MustGetValue(config.ApplicationOwnerKey).AsString(),
		modules:        []Module{},
		configProvider: cfg,
	}

	// TODO: Make this load independently
	svc.scope = cm.Global(true)

	// load config
	cfg.GetValue("", nil).PopulateStruct(&svc.standardConfig)

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
	for _, r := range svc.standardConfig.ServiceRoles {
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

	// if we have an instance, look for a property called "config" and load the service
	// node into that config.
	if instance != nil {

		instanceType := reflect.TypeOf(instance)

		// get the actual value
		for instanceType.Kind() == reflect.Ptr {
			instanceType = instanceType.Elem()
		}

		if configField, found := instanceType.FieldByName("Config"); found {
			configValue := reflect.New(configField.Type)

			// Try to load the service config
			if cfg.GetValue("service", nil).PopulateStruct(configValue.Interface()) {
				instanceValue := reflect.ValueOf(instance).Elem()
				instanceValue.FieldByName("Config").Set(configValue.Elem())
			}
		}

		setupServiceInstance(instance, svc)
		svc.instance = instance
	}

	svc.Scope().Counter("boot").Inc(1)
	return svc
}

func (s *Service) addModule(module Module) error {
	if s.locked {
		return errors.New("ServiceAlreadyStarted")
	}
	s.modules = append(s.modules, module)
	return nil
}

func (s *Service) Name() string {
	return s.standardConfig.ServiceName
}

func (s *Service) Description() string {
	return s.standardConfig.ServiceDescription
}

func (s *Service) Owner() string {
	return s.standardConfig.ServiceOwner
}
func (s *Service) Roles() []string {
	return s.standardConfig.ServiceRoles
}

func (s *Service) Scope() metrics.Scope {
	return s.scope
}
func (s *Service) Modules() []Module {
	mods := make([]Module, len(s.modules))
	copy(mods, s.modules)
	return mods
}

func (s *Service) isRunning() bool {
	return s.closeChan != nil
}

func (s *Service) OnCriticalError(err error) {
	if !s.instance.OnCriticalError(err) {
		s.shutdown(err, "", nil)
	}
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

	if s.instance != nil {
		s.instance.OnShutdown(*s.shutdownReason)
	}
	return true, err
}

// Start runs the serve loop. If Shutdown() was called then the shutdown reason
// will be returned.
func (s *Service) Start(waitForShutdown bool) (<-chan ServiceExit, error) {
	var err error
	s.locked = true
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	if s.inShutdown {
		return nil, errors.New("errShuttingDown")
	} else if s.isRunning() {
		return s.closeChan, nil
	} else {

		if s.instance != nil {
			if err := s.instance.OnInit(s); err != nil {
				return nil, err
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
