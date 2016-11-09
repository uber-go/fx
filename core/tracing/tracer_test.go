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

package tracing

import (
	"testing"

	"go.uber.org/fx/core/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"github.com/uber-go/zap"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
)

var (
	serviceName            = "serviceName"
	logger                 = ulog.Logger()
	scope                  = tally.NoopScope
	emptyJaegerConfig      = &jaegerconfig.Configuration{}
	disabledJaegerConfig   = &jaegerconfig.Configuration{Disabled: true}
	jaegerConfigWithLogger = &jaegerconfig.Configuration{Logger: jaeger.NullLogger}
)

func TestInitGlobalTracer_Simple(t *testing.T) {
	tracer, err := InitGlobalTracer(emptyJaegerConfig, serviceName, logger, scope)
	assert.NotNil(t, tracer)
	assert.NoError(t, err)
}

func TestInitGlobalTracer_Disabled(t *testing.T) {
	tracer, err := InitGlobalTracer(disabledJaegerConfig, serviceName, logger, scope)
	assert.NotNil(t, tracer)
	assert.NoError(t, err)
}

func TestInitGlobalTracer_NoServiceName(t *testing.T) {
	tracer, err := InitGlobalTracer(emptyJaegerConfig, "", logger, scope)
	assert.NotNil(t, err)
	assert.Nil(t, tracer)
}

func TestLoadAppConfig(t *testing.T) {
	jConfig := loadAppConfig(emptyJaegerConfig, logger)
	assert.NotNil(t, jConfig)
	assert.NotNil(t, jConfig.Logger)
}

func TestLoadAppConfig_JaegerConfigWithLogger(t *testing.T) {
	jConfig := loadAppConfig(jaegerConfigWithLogger, logger)
	assert.NotNil(t, jConfig)
	assert.Equal(t, jaeger.NullLogger, jConfig.Logger)
}

func TestLoadAppConfig_NilJaegerConfig(t *testing.T) {
	jConfig := loadAppConfig(nil, logger)
	assert.NotNil(t, jConfig)
	assert.NotNil(t, jConfig.Logger)
}

func TestJaegerLogger(t *testing.T) {
	ulog.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *ulog.TestBuffer) {
		loggerWithZap := ulog.Logger()
		loggerWithZap.SetLogger(zaplogger)
		jLogger := jaegerLogger{log: loggerWithZap}
		jLogger.Infof("info message", "c", "d")
		jLogger.Error("error message")
		assert.Equal(t, []string{
			`{"level":"info","msg":"info message","c":"d"}`,
			`{"level":"error","msg":"error message"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}
