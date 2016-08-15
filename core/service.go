package core

import (
	"reflect"

	"github.com/uber-go/uberfx/core/config"
	cm "github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/uberfx/internal/util"
)

type ServiceState int

const (
	Uninitialized = iota + 1
	Initialized
	Starting
	Running
	Stopping
	Stopped
)

type StateChangeCallback func(ServiceState, ServiceState)

type ServiceOwner interface {
	ServiceHost
	Start(waitForExit bool) (<-chan ServiceExit, error)
	Stop(reason string, exitCode int) error
}

// ServiceInstance is the interface that is implemented by user service/
// code.
type ServiceInstance interface {

	// OnInit will be called after the service has been initialized
	OnInit(service ServiceHost) error

	// OnStateChange is called whenever the service changes
	// states
	OnStateChange(old ServiceState, new ServiceState)

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

func NewService(instance ServiceInstance, options ...ServiceOption) ServiceOwner {

	cfg := config.Global()

	svc := &serviceHost{
		// TODO: get these out of config struct instead
		modules: []Module{},
		serviceCore: serviceCore{
			configProvider: cfg,
			instance:       instance,
			items:          map[string]interface{}{},
		},
	}

	// prepend the default options.
	// TODO: This isn't right.  Do we order the options so we make sure to use
	// the passed in options?
	//
	WithConfiguration(cfg)(svc)

	// load standard config
	svc.configProvider.GetValue("").PopulateStruct(&svc.standardConfig)

	// hash up the roles
	svc.roles = map[string]bool{}
	for _, r := range svc.standardConfig.ServiceRoles {
		svc.roles[r] = true
	}

	WithMetricsScope(cm.Global(true))(svc)

	// Run the rest of the options
	for _, opt := range options {
		if optionErr := opt(svc); optionErr != nil {
			panic(optionErr)
		}
	}

	// if we have an instance, look for a property called "config" and load the service
	// node into that config.
	if instance != nil {

		loadInstanceConfig(svc.configProvider, "service", instance)

		if field, found := util.FindField(instance, nil, reflect.TypeOf((ServiceHost)(nil))); found {
			var sc ServiceHost = &svc.serviceCore
			field.Set(reflect.ValueOf(sc))
		}
		svc.instance = instance
	}

	svc.Metrics().Counter("boot").Inc(1)
	return svc
}
