package tracing

import (
	"time"

	"go.uber.org/fx/core/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
)

// InitGlobalTracer instantiates a new global tracer
func InitGlobalTracer(
	cfg *jaegerconfig.Configuration,
	serviceName string,
	logger ulog.Log,
	scope tally.Scope,
) (opentracing.Tracer, error) {
	appCfg := loadAppConfig(cfg, logger)
	reporter := &jaegerReporter{
		reporter: scope.Reporter(),
	}
	tracer, closer, err := appCfg.New(serviceName, reporter)
	if err != nil {
		return tracer, err
	}
	defer closer.Close()
	opentracing.InitGlobalTracer(tracer)
	return tracer, nil
}

func loadAppConfig(cfg *jaegerconfig.Configuration, logger ulog.Log) *jaegerconfig.Configuration {
	var appCfg *jaegerconfig.Configuration
	if cfg == nil {
		appCfg = &jaegerconfig.Configuration{}
	} else {
		appCfg = cfg
	}
	if appCfg.Logger == nil {
		jaegerlogger := &jaegerLogger{
			log: logger,
		}
		appCfg.Logger = jaegerlogger
	}
	return appCfg
}

type jaegerLogger struct {
	log ulog.Log
}

func (jl *jaegerLogger) Error(msg string) {
	jl.log.Error(msg)
}

func (jl *jaegerLogger) Infof(msg string, args ...interface{}) {
	jl.log.Info(msg, args...)
}

type jaegerReporter struct {
	reporter tally.StatsReporter
}

// TODO: Change to use scope with tally functions to increment/update
func (jr *jaegerReporter) IncCounter(name string, tags map[string]string, value int64) {
	jr.reporter.ReportCounter(name, tags, value)
}

func (jr *jaegerReporter) UpdateGauge(name string, tags map[string]string, value int64) {
	jr.reporter.ReportGauge(name, tags, value)
}

func (jr *jaegerReporter) RecordTimer(name string, tags map[string]string, d time.Duration) {
	jr.reporter.ReportTimer(name, tags, d)
}
