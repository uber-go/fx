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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx/core/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/zap"
)

func TestSimpleLogger(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := Logger()
		log.SetLogger(zaplogger)
		log.Debug("debug message", "a", "b")
		log.Info("info message", "c", "d")
		log.Warn("warn message", "e", "f")
		log.Error("error message", "g", "h")
		assert.Equal(t, []string{
			`{"level":"debug","msg":"debug message","a":"b"}`,
			`{"level":"info","msg":"info message","c":"d"}`,
			`{"level":"warn","msg":"warn message","e":"f"}`,
			`{"level":"error","msg":"error message","g":"h"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}

func TestLoggerWithInitFields(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := Logger("method", "test_method")
		log.SetLogger(zaplogger)

		log.Debug("debug message", "a", "b")
		log.Info("info message", "c", "d")
		log.Warn("warn message", "e", "f")
		log.Error("error message", "g", "h")
		assert.Equal(t, []string{
			`{"level":"debug","msg":"debug message","method":"test_method","a":"b"}`,
			`{"level":"info","msg":"info message","method":"test_method","c":"d"}`,
			`{"level":"warn","msg":"warn message","method":"test_method","e":"f"}`,
			`{"level":"error","msg":"error message","method":"test_method","g":"h"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}

func TestLoggerWithInvalidFields(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := Logger()
		log.SetLogger(zaplogger)
		log.Info("info message", "c")
		log.Info("info message", "c", "d", "e")
		log.DFatal("debug message")
		assert.Equal(t, []string{
			`{"level":"info","msg":"info message","error":"invalid number of arguments"}`,
			`{"level":"info","msg":"info message","error":"invalid number of arguments"}`,
			`{"level":"error","msg":"debug message","error":"invalid number of arguments"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}

func TestFatalsAndPanics(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := Logger()
		log.SetLogger(zaplogger)
		assert.Panics(t, func() { log.Panic("panic level") }, "Expected to panic")
		assert.Equal(t, `{"level":"panic","msg":"panic level"}`, buf.Stripped(), "Unexpected output")
	})

}

func TestConfiguredLogger(t *testing.T) {
	withLogger(t, func(tmpDir string, logFile string) {
		log := Logger()
		txt := false
		cfg := Configuration{
			Level:         "debug",
			Stdout:        false,
			TextFormatter: &txt,
			Verbose:       false,
		}
		log.Configure(cfg)
		zaplogger := log.RawLogger()
		assert.Equal(t, zap.DebugLevel, zaplogger.Level())
	})
}

func TestConfiguredLoggerWithTextFormatter(t *testing.T) {
	withLogger(t, func(tmpDir string, logFile string) {
		log := Logger()
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
		log.Configure(cfg)
		zaplogger := log.RawLogger()
		assert.Equal(t, zap.DebugLevel, zaplogger.Level())
	})
}

func TestConfiguredLoggerWithStdout(t *testing.T) {
	withLogger(t, func(tmpDir string, logFile string) {
		log := Logger()
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
		log.Configure(cfg)
		zaplogger := log.RawLogger()
		assert.Equal(t, zap.DebugLevel, zaplogger.Level())
	})
}

func withLogger(t *testing.T, f func(string, string)) {
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
	Configure(cfg)
	f(tmpDir, logFile)
}

func TestDefaultPackageLogger(t *testing.T) {
	withLogger(t, func(tmpDir string, logFile string) {
		logger := Logger()
		zaplogger := logger.RawLogger()
		assert.Equal(t, zap.ErrorLevel, zaplogger.Level())
		logger.SetLevel(zap.WarnLevel)
		assert.Equal(t, zap.WarnLevel, zaplogger.Level())
	})
}

func TestDefaultLoggingWithInitFields(t *testing.T) {
	withLogger(t, func(tmpDir string, logFile string) {
		logger := Logger("a", "b")
		logger.Error("test log")
		content, err := ioutil.ReadFile(filepath.Join(tmpDir, logFile))
		require.NoError(t, err)
		assert.True(t, strings.Contains(string(content), "test log"))
		assert.True(t, strings.Contains(string(content), `"a":"b"`))
	})
}

type marshalObject struct {
	Data string `json:"data"`
}

func (m *marshalObject) MarshalLog(kv zap.KeyValue) error {
	kv.AddString("Data", m.Data)
	return nil
}

func TestFieldConversion(t *testing.T) {
	log := Logger()
	base := log.(*baselogger)

	assert.Equal(t, zap.Bool("a", true), base.fieldsConversion("a", true)[0])
	assert.Equal(t, zap.Float64("a", 5.5), base.fieldsConversion("a", 5.5)[0])
	assert.Equal(t, zap.Int("a", 10), base.fieldsConversion("a", 10)[0])
	assert.Equal(t, zap.Int64("a", int64(10)), base.fieldsConversion("a", int64(10))[0])
	assert.Equal(t, zap.Uint("a", uint(10)), base.fieldsConversion("a", uint(10))[0])
	assert.Equal(t, zap.Uintptr("a", uintptr(0xa)), base.fieldsConversion("a", uintptr(0xa))[0])
	assert.Equal(t, zap.Uint64("a", uint64(10)), base.fieldsConversion("a", uint64(10))[0])
	assert.Equal(t, zap.String("a", "xyz"), base.fieldsConversion("a", "xyz")[0])
	assert.Equal(t, zap.Time("a", time.Unix(0, 0)), base.fieldsConversion("a", time.Unix(0, 0))[0])
	assert.Equal(t, zap.Duration("a", time.Microsecond), base.fieldsConversion("a", time.Microsecond)[0])
	dt := &marshalObject{Data: "value"}
	assert.Equal(t, zap.Marshaler("a", &marshalObject{"value"}), base.fieldsConversion("a", dt)[0])
	ip := net.ParseIP("1.2.3.4")
	assert.Equal(t, zap.Stringer("ip", ip), base.fieldsConversion("ip", ip)[0])
	assert.Equal(t, zap.Object("a", []int{1, 2}), base.fieldsConversion("a", []int{1, 2})[0])
	err := fmt.Errorf("test error")
	assert.Equal(t, zap.Error(err), base.fieldsConversion("error", err)[0])

}

func TestRawLogger(t *testing.T) {
	log := Logger()
	assert.NotNil(t, log.RawLogger())
}
