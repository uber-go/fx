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
	"errors"

	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog/sentry"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Configuration defines the desired logging options.
type Configuration struct {
	zap.Config

	Sentry *sentry.Configuration `yaml:"sentry"`
}

// Configure initializes logging configuration struct from config provider
func (c *Configuration) Configure(cfg config.Value) error {
	// TODO: Fix after GFM-415
	// Uhhh... this process is not the most elegant.
	//
	// Because log.Configuration embeds zap, the Populate
	// does not work properly as it's unable to serialize fields directly
	// into the embedded struct, so inner struct has to be treated as a
	// separate object
	//
	// first, use the default zap configuration
	zapCfg := DefaultConfiguration().Config

	// override the embedded zap.Config stuct from config
	if err := cfg.Populate(&zapCfg); err != nil {
		return errors.New("unable to parse logging config")
	}

	// use the overriden zap config
	c.Config = zapCfg

	// override any remaining things fom config, i.e. Sentry
	if err := cfg.Populate(&c); err != nil {
		return errors.New("unable to parse logging config")
	}

	return nil
}

// Build constructs a *zap.Logger with the configured parameters.
func (c Configuration) Build(opts ...zap.Option) (*zap.Logger, error) {
	logger, err := c.Config.Build(opts...)
	if err != nil || c.Sentry == nil {
		// If there's an error or there's no Sentry config, we don't need to do
		// anything but delegate.
		return logger, err
	}
	sentry, err := c.Sentry.Build()
	if err != nil {
		return logger, err
	}
	return logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentry)
	})), nil
}

// DefaultConfiguration returns a fallback configuration for applications that
// don't explicitly configure logging.
func DefaultConfiguration() Configuration {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}

	return Configuration{
		Config: cfg,
	}
}

// SetLogger uses the provided logger to replace zap's global loggers and
// hijack output from the standard library's "log" package. It returns a
// function to undo these changes.
func SetLogger(log *zap.Logger) func() {
	undoGlobals := zap.ReplaceGlobals(log)
	undoHijack := zap.RedirectStdLog(log)
	return func() {
		undoGlobals()
		undoHijack()
	}
}
