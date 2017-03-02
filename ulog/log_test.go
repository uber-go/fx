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
	"context"
	"testing"

	"go.uber.org/fx/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	_logyaml = []byte(`
logging:
  stdout: true
`)

	_sentryyaml = []byte(`
logging:
  sentry:
    dsn:
    trace:
      disabled: true
      skip_frames: 10
      context_lines: 15
`)
)

func TestDefaultLogger(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfiguration()
	assert.NotNil(t, cfg)
	assert.Equal(t, []string{"stdout"}, cfg.OutputPaths)
}

func TestSetLogger(t *testing.T) {
	t.Parallel()
	zaplogger := zapcore.NewNopCore()
	defer SetLogger(zap.New(zaplogger))()

	log := Logger(context.Background())
	assert.Equal(t, zaplogger, log.Core())

	sugarlog := Sugar(context.Background())
	assert.Equal(t, zaplogger, sugarlog.Desugar().Core())
}

func TestConfigureLogger(t *testing.T) {
	var logConfig Configuration
	cfg := config.NewYAMLProviderFromBytes(_logyaml).Get("logging")
	err := logConfig.Configure(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logConfig)
	logger, err := logConfig.Build()
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestConfigureSentryLogger(t *testing.T) {
	var logConfig Configuration
	cfg := config.NewYAMLProviderFromBytes(_sentryyaml).Get("logging")
	err := logConfig.Configure(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logConfig.Sentry)
	assert.Equal(t, "", logConfig.Sentry.DSN)
	assert.Equal(t, bool(true), logConfig.Sentry.Trace.Disabled)
	assert.Equal(t, 10, *logConfig.Sentry.Trace.SkipFrames)
	assert.Equal(t, 15, *logConfig.Sentry.Trace.ContextLines)

	logger, err := logConfig.Build()
	require.NoError(t, err)
	assert.NotNil(t, logger)
}
