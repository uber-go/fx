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
	"time"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/go-validator/validator"
	"github.com/pkg/errors"
)

func (svc *serviceCore) setupLogging(cfg config.Provider) {
	if svc.log == nil {
		logBuilder := ulog.Builder()
		// load and configure logging
		err := svc.configProvider.Get("logging").PopulateStruct(&svc.logConfig)
		if err != nil {
			ulog.Logger().Info("Logging configuration is not provided, setting to default logger", "error", err)
		}
		svc.log = logBuilder.WithConfiguration(svc.logConfig).Build()
	} else {
		svc.log.Debug("Using custom log provider due to service.WithLogger option")
	}
}

func (svc *serviceCore) setupStandardConfig(cfg config.Provider) error {
	if err := cfg.Get(config.Root).PopulateStruct(&svc.standardConfig); err != nil {
		return errors.Wrap(err, "unable to load standard configuration")
	}

	if errs := validator.Validate(svc.standardConfig); errs != nil {
		svc.Logger().Error("Invalid service configuration", "error", errs)
		return errors.Wrap(errs, "service configuration failed validation")
	}
	return nil
}

func (svc *serviceCore) setupMetrics() {
	if svc.Metrics() == nil {
		svc.metrics, svc.statsReporter, svc.metricsCloser = metrics.RootScope(svc)
		metrics.Freeze()
	}
}

func (svc *serviceCore) setupRuntimeMetricsCollector(cfg config.Provider) error {
	if svc.RuntimeMetricsCollector() != nil {
		return nil
	}

	var runtimeMetricsConfig metrics.RuntimeConfig
	err := cfg.Get("metrics.runtime").PopulateStruct(&runtimeMetricsConfig)
	if err != nil {
		return errors.Wrap(err, "unable to load runtime metrics configuration")
	}
	svc.runtimeCollector = metrics.StartCollectingRuntimeMetrics(
		svc.metrics.SubScope("runtime"), time.Second, runtimeMetricsConfig,
	)
	return nil
}

func (svc *serviceCore) setupTracer(cfg config.Provider) error {
	if svc.Tracer() != nil {
		return nil
	}
	if err := cfg.Get("tracing").PopulateStruct(&svc.tracerConfig); err != nil {
		return errors.Wrap(err, "unable to load tracing configuration")
	}
	tracer, closer, err := tracing.InitGlobalTracer(
		&svc.tracerConfig,
		svc.standardConfig.ServiceName,
		svc.log,
		svc.statsReporter,
	)
	if err != nil {
		return errors.Wrap(err, "unable to initialize global tracer")
	}
	svc.tracer = tracer
	svc.tracerCloser = closer
	return nil
}

func (svc *serviceCore) setupObserver(cfg config.Provider) {
	if svc.observer != nil {
		loadInstanceConfig(svc.configProvider, "service", svc.observer)

		if shc, ok := svc.observer.(SetContainerer); ok {
			shc.SetContainer(svc)
		}
	}
}

func (svc *serviceCore) setupAuthClient(cfg config.Provider) {
	if svc.authClient != nil {
		return
	}
	auth.SetupClient(svc)
	svc.authClient = auth.Instance()
}
