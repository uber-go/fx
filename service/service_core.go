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
	"io"
	"reflect"
	"sync"
	"time"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/internal/util"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/go-validator/validator"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

const instanceConfigName = "ServiceConfig"

type metricsCore struct {
	metrics          tally.Scope
	statsReporter    tally.CachedStatsReporter
	metricsCloser    io.Closer
	runtimeCollector *metrics.RuntimeCollector
	versionEmitter   *versionMetricsEmitter
}

func (mc *metricsCore) Metrics() tally.Scope {
	return mc.metrics
}

func (mc *metricsCore) MetricsCloser() io.Closer {
	return mc.metricsCloser
}

func (mc *metricsCore) RuntimeMetricsCollector() *metrics.RuntimeCollector {
	return mc.runtimeCollector
}

type tracerCore struct {
	tracer       opentracing.Tracer
	tracerCloser io.Closer
	tracerConfig jaegerconfig.Configuration
}

func (tc *tracerCore) Tracer() opentracing.Tracer {
	return tc.tracer
}

type serviceConfig struct {
	Name        string   `yaml:"name" validate:"nonzero"`
	Owner       string   `yaml:"owner"  validate:"nonzero"`
	Description string   `yaml:"description"`
	Roles       []string `yaml:"roles"`
}

// Implements Host interface
type serviceCore struct {
	metricsCore
	tracerCore
	authClient     auth.Client
	configProvider config.Provider
	logConfig      ulog.Configuration
	observer       Observer
	roles          []string
	scopeMux       sync.Mutex
	standardConfig serviceConfig
	state          State
	moduleName     string
	graph          *dig.Graph
}

var _ Host = &serviceCore{}

func (s *serviceCore) AuthClient() auth.Client {
	return s.authClient
}

func (s *serviceCore) Name() string {
	return s.standardConfig.Name
}

func (s *serviceCore) ModuleName() string {
	return s.moduleName
}

func (s *serviceCore) Description() string {
	return s.standardConfig.Description
}

// ServiceOwner is a string in config.
// Manager is also a struct that embeds Host
func (s *serviceCore) Owner() string {
	return s.standardConfig.Owner
}

func (s *serviceCore) State() State {
	return s.state
}

func (s *serviceCore) Roles() []string {
	return s.standardConfig.Roles
}

func (s *serviceCore) Observer() Observer {
	return s.observer
}

func (s *serviceCore) Config() config.Provider {
	return s.configProvider
}

func (s *serviceCore) Graph() *dig.Graph {
	return s.graph
}

func (s *serviceCore) setupLogging() error {
	cfg := s.configProvider.Get("logging")
	if cfg.HasValue() {
		if err := s.logConfig.Configure(cfg); err != nil {
			return errors.Wrap(err, "failed to initialize logging from config")
		}
	} else {
		// if no config - default to the regular one
		s.logConfig = ulog.DefaultConfiguration()
	}

	logger, err := s.logConfig.Build(zap.Hooks(ulog.Metrics(s.metrics)))
	if err != nil {
		return errors.Wrap(err, "failed to build the logger")
	}

	// TODO(glib): SetLogger returns a deferral to clean up global log which is not used
	ulog.SetLogger(logger)

	return nil
}

func (s *serviceCore) setupStandardConfig() error {
	if err := s.configProvider.Get(config.Root).Populate(&s.standardConfig); err != nil {
		return errors.Wrap(err, "unable to load standard configuration")
	}

	if errs := validator.Validate(s.standardConfig); errs != nil {
		zap.L().Error("Invalid service configuration", zap.Error(errs))
		return errors.Wrap(errs, "service configuration failed validation")
	}
	return nil
}

func (s *serviceCore) setupMetrics() {
	if s.Metrics() == nil {
		s.metrics, s.statsReporter, s.metricsCloser = metrics.RootScope(s)
		metrics.Freeze()
	}
}

func (s *serviceCore) setupRuntimeMetricsCollector() error {
	if s.RuntimeMetricsCollector() != nil {
		return nil
	}

	var runtimeMetricsConfig metrics.RuntimeConfig
	err := s.configProvider.Get("metrics.runtime").Populate(&runtimeMetricsConfig)
	if err != nil {
		return errors.Wrap(err, "unable to load runtime metrics configuration")
	}
	s.runtimeCollector = metrics.StartCollectingRuntimeMetrics(
		s.metrics.SubScope("runtime"), time.Second, runtimeMetricsConfig,
	)
	return nil
}

func (s *serviceCore) setupVersionMetricsEmitter() {
	s.versionEmitter = newVersionMetricsEmitter(s.metrics)
	s.versionEmitter.start()
}

func (s *serviceCore) setupTracer() error {
	if s.Tracer() != nil {
		return nil
	}
	if err := s.configProvider.Get("tracing").Populate(&s.tracerConfig); err != nil {
		return errors.Wrap(err, "unable to load tracing configuration")
	}
	tracer, closer, err := tracing.InitGlobalTracer(
		&s.tracerConfig,
		s.standardConfig.Name,
		zap.L(),
		s.metrics,
	)
	if err != nil {
		return errors.Wrap(err, "unable to initialize global tracer")
	}
	s.tracer = tracer
	s.tracerCloser = closer
	return nil
}

func (s *serviceCore) setupObserver() {
	if s.observer != nil {
		loadInstanceConfig(s.configProvider, "service", s.observer)

		if shc, ok := s.observer.(SetContainerer); ok {
			shc.SetContainer(s)
		}
	}
}

func (s *serviceCore) setupAuthClient() {
	if s.authClient != nil {
		return
	}
	s.authClient = auth.Load(s)
}

func loadInstanceConfig(cfg config.Provider, key string, instance interface{}) bool {
	fieldName := instanceConfigName
	if field, found := util.FindField(instance, &fieldName, nil); found {

		configValue := reflect.New(field.Type())

		// Try to load the service config
		err := cfg.Get(key).Populate(configValue.Interface())
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
