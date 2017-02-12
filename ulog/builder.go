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

package ulog

import (
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog/metrics"
	"go.uber.org/fx/ulog/sentry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var defaultEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "msg",
	StacktraceKey:  "stacktrace",
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.EpochTimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
}

// Configuration for logging with UberFx
type Configuration struct {
	Level   string
	File    *FileConfiguration
	Stdout  bool
	Verbose bool

	// Do not automatically emit metrics for logging counts
	DisableMetrics bool

	Sentry *sentry.Configuration

	prefixWithFileLine *bool `yaml:"prefix_with_fileline"`
	TextFormatter      *bool // use TextFormatter (default json)
}

// FileConfiguration describes the properties needed to log to a file
type FileConfiguration struct {
	Enabled   bool
	Directory string
	FileName  string
}

// LogBuilder is the struct containing logger
type LogBuilder struct {
	log        *zap.Logger
	logConfig  Configuration
	sentryHook *sentry.Hook
	scope      tally.Scope
}

// Builder creates an empty builder for building ulog.Log object
func Builder() *LogBuilder {
	return &LogBuilder{}
}

// New instance of ulog.Log is returned with the default setup
func New() (Log, error) {
	return Builder().Build()
}

// WithConfiguration sets up configuration for the log builder
func (lb *LogBuilder) WithConfiguration(logConfig Configuration) *LogBuilder {
	lb.logConfig = logConfig
	return lb
}

// WithScope sets up configuration for the log builder
func (lb *LogBuilder) WithScope(s tally.Scope) *LogBuilder {
	lb.scope = s
	return lb
}

// SetLogger allows users to set their own initialized logger to work with ulog APIs
func (lb *LogBuilder) SetLogger(zap *zap.Logger) *LogBuilder {
	lb.log = zap
	return lb
}

// WithSentryHook allows users to manually configure the sentry hook
func (lb *LogBuilder) WithSentryHook(hook *sentry.Hook) *LogBuilder {
	lb.sentryHook = hook
	return lb
}

// Build the ulog logger for use
func (lb *LogBuilder) Build() (Log, error) {
	// When setLogger is called, we will always use logger that has been set
	if lb.log != nil {
		return &baseLogger{
			log: lb.log,
		}, nil
	}

	var log *zap.Logger
	var err error
	if config.IsDevelopmentEnv() {
		log, err = lb.devLogger()
	} else {
		log, err = lb.Configure()
	}
	if err != nil {
		return nil, err
	}
	// TODO(pedge): there's no point in setting this anymore I think
	lb.log = log

	// TODO(glib): document that yaml configuration takes precedence or
	// make the situation better so yaml overrides only the DSN
	if lb.logConfig.Sentry != nil {
		if len(lb.logConfig.Sentry.DSN) > 0 {
			hook, err := sentry.Configure(*lb.logConfig.Sentry)
			if err != nil {
				// TODO(pedge): it would be nicer to return the error
				log.Warn("Sentry creation failed with error", zap.Error(err))
			} else {
				lb.sentryHook = hook
			}
		}
	}

	return &baseLogger{
		log: log,
		sh:  lb.sentryHook,
	}, nil
}

func (lb *LogBuilder) devLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

// Configure Log object with the provided log.Configuration
func (lb *LogBuilder) Configure() (*zap.Logger, error) {
	levelEnabler := zap.DebugLevel
	if !lb.logConfig.Verbose {
		// TODO(pedge): not set anymore
		//lb.log.Info("Setting log level", zap.String("level", lb.logConfig.Level))
		var lv zapcore.Level
		if err := lv.UnmarshalText([]byte(lb.logConfig.Level)); err != nil {
			return nil, errors.Wrap(err, "cannot parse log level")
		}
		levelEnabler = lv
	}

	var writeSyncer zapcore.WriteSyncer
	var err error
	if lb.logConfig.File == nil || !lb.logConfig.File.Enabled {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		writeSyncer, err = lb.fileOutput(lb.logConfig.File, lb.logConfig.Stdout, lb.logConfig.Verbose)
		if err != nil {
			return nil, err
		}
	}

	encoder := zapcore.NewJSONEncoder(defaultEncoderConfig)
	if lb.logConfig.TextFormatter != nil && *lb.logConfig.TextFormatter {
		encoder = zapcore.NewConsoleEncoder(defaultEncoderConfig)
	}

	facility := zapcore.WriterFacility(encoder, writeSyncer, levelEnabler)
	if lb.scope != nil && !lb.logConfig.DisableMetrics {
		sub := lb.scope.SubScope("logging")
		facility = zapcore.Hooked(facility, metrics.Hook(sub))
	}

	return zap.New(facility), nil
}

func (lb *LogBuilder) fileOutput(cfg *FileConfiguration, stdout bool, verbose bool) (zapcore.WriteSyncer, error) {
	fileLoc := path.Join(cfg.Directory, cfg.FileName)
	// TODO(pedge): not set anymore
	//lb.log.Debug("adding log file output to zap")
	if err := os.MkdirAll(cfg.Directory, os.FileMode(0755)); err != nil {
		return nil, errors.Wrap(err, "failed to create log directory")
	}
	file, err := os.OpenFile(fileLoc, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open log file for writing")
	}
	// TODO(pedge): not set anymore
	//lb.log.Debug("Logfile created successfully", zap.String("filename", fileLoc))
	if verbose || stdout {
		return zapcore.AddSync(io.MultiWriter(os.Stdout, file)), nil
	}
	return zapcore.AddSync(file), nil
}
