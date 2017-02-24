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

package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

var (
	_serviceName          = "serviceName"
	_scope                = tally.NoopScope
	_emptyJaegerConfig    = &config.Configuration{}
	_disabledJaegerConfig = &config.Configuration{Disabled: true}
)

func TestInitGlobalTracer_Simple(t *testing.T) {
	tracer, closer, err := InitGlobalTracer(
		_emptyJaegerConfig, _serviceName, zap.L(), _scope,
	)
	defer closer.Close()
	assert.NotNil(t, tracer)
	assert.NotNil(t, closer)
	assert.NoError(t, err)
}

func TestInitGlobalTracer_Disabled(t *testing.T) {
	tracer, closer, err := InitGlobalTracer(
		_disabledJaegerConfig, _serviceName, zap.L(), _scope,
	)
	defer closer.Close()
	assert.NotNil(t, tracer)
	assert.NotNil(t, closer)
	assert.NoError(t, err)
}

func TestInitGlobalTracer_NoServiceName(t *testing.T) {
	tracer, closer, err := InitGlobalTracer(_emptyJaegerConfig, "", zap.L(), _scope)
	assert.Error(t, err)
	assert.Nil(t, tracer)
	assert.Nil(t, closer)
}

func TestLoadAppConfig(t *testing.T) {
	jConfig := loadAppConfig(_emptyJaegerConfig)
	assert.NotNil(t, jConfig)
}

func TestLoadAppConfig_NilJaegerConfig(t *testing.T) {
	jConfig := loadAppConfig(nil)
	assert.NotNil(t, jConfig)
}
