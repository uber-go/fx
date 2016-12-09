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

package ulog

import (
	"io"
	"os"
	"path"

	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog/sentry"

	"github.com/uber-go/zap"
)

// Configuration for logging with UberFx
type Configuration struct {
	Level   string
	File    *FileConfiguration
	Stdout  bool
	Verbose bool

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
	log        zap.Logger
	logConfig  Configuration
	sentryHook *sentry.Hook
}

// Builder creates an empty builder for building ulog.Log object
func Builder() *LogBuilder {
	return &LogBuilder{}
}

// New instance of ulog.Log is returned with the default setup
func New() Log {
	return Builder().Build()
}

// WithConfiguration sets up configuration for the log builder
func (lb *LogBuilder) WithConfiguration(logConfig Configuration) *LogBuilder {
	lb.logConfig = logConfig
	return lb
}

// SetLogger allows users to set their own initialized logger to work with ulog APIs
func (lb *LogBuilder) SetLogger(zap zap.Logger) *LogBuilder {
	lb.log = zap
	return lb
}

// WithSentryHook lalalal
func (lb *LogBuilder) WithSentryHook(hook *sentry.Hook) *LogBuilder {
	lb.sentryHook = hook
	return lb
}

// Build the ulog logger for use
func (lb *LogBuilder) Build() Log {
	var log zap.Logger

	// When setLogger is called, we will always use logger that has been set
	if lb.log != nil {
		return &baseLogger{
			log: lb.log,
		}
	}

	if config.IsDevelopment() {
		log = lb.devLogger()
	} else {
		log = lb.Configure()
	}

	return &baseLogger{
		log: log,
		sh:  lb.sentryHook,
	}
}

func (lb *LogBuilder) devLogger() zap.Logger {
	return zap.New(zap.NewTextEncoder(), zap.DebugLevel)
}

func (lb *LogBuilder) defaultLogger() zap.Logger {
	return zap.New(zap.NewJSONEncoder(), zap.InfoLevel, zap.Output(zap.AddSync(os.Stdout)))
}

// Configure Log object with the provided log.Configuration
func (lb *LogBuilder) Configure() zap.Logger {
	lb.log = lb.defaultLogger()

	var options []zap.Option
	if lb.logConfig.Verbose {
		options = append(options, zap.DebugLevel)
	} else {
		lb.log.Info("Setting log level", zap.String("level", lb.logConfig.Level))
		var lv zap.Level
		err := lv.UnmarshalText([]byte(lb.logConfig.Level))
		if err != nil {
			lb.log.Debug(
				"Cannot parse log level. Setting to Debug as default",
				zap.String("level", lb.logConfig.Level),
			)
		} else {
			options = append(options, lv)
		}
	}

	if lb.logConfig.File == nil || !lb.logConfig.File.Enabled {
		options = append(options, zap.Output(zap.AddSync(os.Stdout)))
	} else {
		options = append(options, zap.Output(lb.fileOutput(lb.logConfig.File, lb.logConfig.Stdout, lb.logConfig.Verbose)))
	}

	if lb.logConfig.TextFormatter != nil && *lb.logConfig.TextFormatter {
		return zap.New(zap.NewTextEncoder(), options...)
	}
	return zap.New(zap.NewJSONEncoder(), options...)
}

func (lb *LogBuilder) fileOutput(cfg *FileConfiguration, stdout bool, verbose bool) zap.WriteSyncer {
	fileLoc := path.Join(cfg.Directory, cfg.FileName)
	lb.log.Debug("adding log file output to zap")
	err := os.MkdirAll(cfg.Directory, os.FileMode(0755))
	if err != nil {
		lb.log.Fatal("Failed to create log directory: ", zap.Error(err))
	}
	file, err := os.OpenFile(fileLoc, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if err != nil {
		lb.log.Fatal("Failed to open log file for writing.", zap.Error(err))
	}
	lb.log.Debug("Logfile created successfully", zap.String("filename", fileLoc))
	if verbose || stdout {
		return zap.AddSync(io.MultiWriter(os.Stdout, file))
	}
	return zap.AddSync(file)
}
