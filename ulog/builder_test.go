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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/testutils/env"
	"go.uber.org/fx/ulog/sentry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"github.com/uber-go/zap"
	"go.uber.org/fx/testutils/metrics"
)

func TestConfiguredLogger(t *testing.T) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		txt := false
		cfg := Configuration{
			Level:         "debug",
			Stdout:        false,
			TextFormatter: &txt,
			Verbose:       false,
		}
		log := builder.WithConfiguration(cfg).Build()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
	})
}

func TestConfiguredLoggerWithTextFormatter(t *testing.T) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		txt := true
		cfg := Configuration{
			Level:         "debug",
			Stdout:        false,
			TextFormatter: &txt,
			Verbose:       false,
			File: &FileConfiguration{
				Directory: tmpDir,
				FileName:  logFile,
				Enabled:   true,
			},
		}
		log := Builder().WithConfiguration(cfg).Build()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
	})
}

func TestConfiguredLoggerWithTextFormatter_NonDev(t *testing.T) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		txt := true
		log := Builder().WithConfiguration(Configuration{
			Level:         "debug",
			TextFormatter: &txt,
		}).Build()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
	})
}

func TestConfiguredLoggerWithStdout(t *testing.T) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		txt := false
		cfg := Configuration{
			Stdout:        true,
			TextFormatter: &txt,
			Verbose:       true,
			File: &FileConfiguration{
				Enabled:   true,
				Directory: tmpDir,
				FileName:  logFile,
			},
		}
		log := Builder().WithConfiguration(cfg).Build()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
	})
}

func withLogger(t *testing.T, f func(*LogBuilder, string, string)) {
	defer env.Override(t, config.EnvironmentKey(), "madeup")()
	tmpDir, err := ioutil.TempDir("", "default_log")
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir), "should be able to delete tempdir")
	}()
	require.NoError(t, err)

	tmpFile, err := ioutil.TempFile(tmpDir, "temp_log.txt")
	require.NoError(t, err)
	logFile, err := filepath.Rel(tmpDir, tmpFile.Name())
	require.NoError(t, err)
	txt := false
	cfg := Configuration{
		Level:         "error",
		Stdout:        false,
		TextFormatter: &txt,
		Verbose:       false,
		File: &FileConfiguration{
			Enabled:   true,
			Directory: tmpDir,
			FileName:  logFile,
		},
	}
	builder := Builder().WithConfiguration(cfg)
	f(builder, tmpDir, logFile)
}

func TestDefaultPackageLogger(t *testing.T) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		defer env.Override(t, config.EnvironmentKey(), "development")()
		log := New()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
	})
}

func TestConfiguredLoggerWithSentrySuccessful(t *testing.T) {
	testSentry(t, "https://u:p@example.com/sentry/1", true)
}

func TestConfiguredLoggerWithSentryError(t *testing.T) {
	testSentry(t, "", false)
	testSentry(t, "invalid_dsn", false)
}

func TestMetricsHook(t *testing.T) {
	// TODO(glib): logging init needs to be fixed
	// Any tests that run in development environment can't configure the logger,
	// it just creates the default one...
	defer env.Override(t, config.EnvironmentKey(), "wat?")()

	s, r := metrics.NewTestScope()
	l := Builder().WithScope(s).Build()

	r.CountersWG.Add(1)
	l.Warn("Warning log!")
	r.CountersWG.Wait()

	assert.Equal(t, 1, len(r.Counters))
}

func TestLoggingMetricsDisabled(t *testing.T) {
	testCases := []struct {
		name           string
		disableMetrics bool
		optsLen        int
	}{
		{"Disabled", true, 1},
		{"Enabled", false, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logCfg := Configuration{DisableMetrics: tc.disableMetrics}
			builder := Builder().WithConfiguration(logCfg).WithScope(tally.NoopScope)
			builder.log = builder.defaultLogger()
			opts := builder.zapOptions()
			assert.Equal(t, tc.optsLen, len(opts))
		})
	}
}

func testSentry(t *testing.T, dsn string, isValid bool) {
	withLogger(t, func(builder *LogBuilder, tmpDir string, logFile string) {
		txt := false
		cfg := Configuration{
			Level:         "debug",
			Stdout:        false,
			TextFormatter: &txt,
			Verbose:       false,
			Sentry:        &sentry.Configuration{DSN: dsn},
		}
		logBuilder := builder.WithConfiguration(cfg)
		log := logBuilder.Build()
		zapLogger := log.Typed()
		assert.True(t, zapLogger.Check(zap.DebugLevel, "").OK())
		if isValid {
			assert.NotNil(t, logBuilder.sentryHook)
		} else {
			assert.Nil(t, logBuilder.sentryHook)
		}
	})
}
