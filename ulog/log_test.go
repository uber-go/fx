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
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx/internal"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/ulog/sentry"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/zap"
	"github.com/uber/jaeger-client-go"
)

func TestZapSink(t *testing.T) {
	l, s := TestingLogger()
	l.Info("This is my message", "And this is", "My field")
	assert.Equal(t, 1, len(s.Logs()))

	e := s.Logs()[0]
	assert.Equal(t, "This is my message", e.Msg)
	assert.Contains(t, e.Fields, zap.String("And this is", "My field"))
}

func TestLogger_SetLogger(t *testing.T) {
	SetLogger(logger())
	assert.NotNil(t, _std)
}

func TestContext_LoggerAccess(t *testing.T) {
	ctx := NewLogContext(context.Background(), nil)
	assert.NotNil(t, ctx)
	assert.NotNil(t, Logger(ctx))
	assert.NotNil(t, ctx.Value(internal.ContextLogger))
}

func TestWithTracingAware(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		// Create in-memory logger and jaeger tracer
		loggerWithZap := Builder().SetLogger(zapLogger).Build()
		tracer, closer := jaeger.NewTracer(
			"serviceName", jaeger.NewConstSampler(true), jaeger.NewNullReporter(),
		)
		defer closer.Close()
		span := tracer.StartSpan("opName")
		ctx := context.WithValue(context.Background(), internal.ContextLogger, loggerWithZap)
		ctx = WithTracingAware(ctx, span)
		Logger(ctx).Info("Testing context aware logger")
		assert.True(t, len(buf.Lines()) > 0)
		for _, line := range buf.Lines() {
			assert.Contains(t, line, "traceID")
			assert.Contains(t, line, "spanID")
		}
	})
}

func TestSimpleLogger(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		log := Builder().SetLogger(zapLogger).Build()

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
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		log := Builder().SetLogger(zapLogger).Build().With("method", "test_method")
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
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		log := Builder().SetLogger(zapLogger).Build()
		log.Info("info message", "c")
		log.Info("info message", "c", "d", "e")
		log.DPanic("debug message")
		assert.Equal(t, []string{
			`{"level":"info","msg":"info message","error":"expected even number of arguments"}`,
			`{"level":"info","msg":"info message","error":"expected even number of arguments"}`,
			`{"level":"dpanic","msg":"debug message"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}

func TestFatalsAndPanics(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		log := Builder().SetLogger(zapLogger).Build()
		assert.Panics(t, func() { log.Panic("panic level") }, "Expected to panic")
		assert.Equal(t, `{"level":"panic","msg":"panic level"}`, buf.Stripped(), "Unexpected output")
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
	log := New()
	base := log.(*baseLogger)

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

func TestTyped(t *testing.T) {
	log := New()
	assert.NotNil(t, log.Typed())
}

func TestLogger(t *testing.T) {
	log := logger()
	assert.NotNil(t, log.Typed())
}

func TestSentryHook(t *testing.T) {
	c := &sentry.MemCapturer{}
	h, err := sentry.New("")
	assert.NoError(t, err, "Need to be able to create a sentry hook")
	h.Capturer = c

	l := Builder().WithSentryHook(h).Build()

	l.Error("you work, yea?", "key", 123)
	l.Info("this should not be sent, right?", "key", "val")

	assert.Equal(t, 1, len(c.Packets))
	p := c.Packets[0]
	assert.Equal(t, "you work, yea?", p.Message)
}

func TestSentryHookDoesNotMutatePrevious(t *testing.T) {
	h, err := sentry.New("")
	defer h.Close()
	assert.NoError(t, err)

	l := Builder().WithSentryHook(h).Build().(*baseLogger)
	assert.Equal(t, make(map[string]interface{}), l.sh.Fields())

	l2 := l.With("key", "value").(*baseLogger)
	assert.Equal(t, map[string]interface{}{}, l.sh.Fields())
	assert.NotNil(t, l2.sh)
	assert.NotNil(t, l2.sh.Fields())
	assert.Equal(t, map[string]interface{}{"key": "value"}, l2.sh.Fields())
}

func TestStackTraceLogger(t *testing.T) {
	t.Parallel()
	testutils.WithInMemoryLogger(t, nil, func(zapLogger zap.Logger, buf *testutils.TestBuffer) {
		log := Builder().SetLogger(zapLogger).Build()
		err1 := errors.New("for sure")
		err2 := errors.Wrap(err1, "it's a trap")
		log.Error("error message", "error", err2)
		line := buf.Lines()[0]
		assert.Contains(t, line, "TestStackTraceLogger.func1", "Missing first function")
		assert.Contains(t, line, filepath.Join("ulog", "log_test.go"), "Missing source")
		assert.Regexp(t, `log_test.go:\d+`, line, "Missing line numbers")

		assert.Contains(t, line, "testutils.WithInMemoryLogger", "Missing second function")
		assert.Contains(t, line, filepath.Join("testutils", "in_memory_log.go"), "Missing source")
		assert.True(t, strings.Index(line, "log_test.go") < strings.Index(line, "in_memory_log.go"))

		assert.Contains(t, line, "it's a trap: for sure")
		assert.Equal(t, 1, len(buf.Lines()))
	})
}
