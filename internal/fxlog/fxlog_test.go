// Copyright (c) 2020 Uber Technologies, Inc.
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

package fxlog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/internal/testutil"
	"go.uber.org/zap/zaptest"
)

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() {
		DefaultLogger(testutil.WriteSyncer{T: t})
	})
}

type testLogSpy struct {
	testing.TB
	Messages []string
}

func newTestLogSpy(tb testing.TB) *testLogSpy {
	return &testLogSpy{TB: tb}
}

func (t *testLogSpy) Logf(format string, args ...interface{}) {
	// Log messages are in the format,
	//
	//   2017-10-27T13:03:01.000-0700	DEBUG	your message here	{data here}
	//
	// We strip the first part of these messages because we can't really test
	// for the timestamp from these tests.
	m := fmt.Sprintf(format, args...)
	m = m[strings.IndexByte(m, '\t')+1:]
	t.Messages = append(t.Messages, m)
	t.TB.Log(m)
}

func (t *testLogSpy) AssertMessages(msgs ...string) {
	assert.Equal(t.TB, msgs, t.Messages, "logged messages did not match")
}

func (t *testLogSpy) Reset() {
	t.Messages = t.Messages[:0]
}

func TestZapLogger(t *testing.T) {
	t.Parallel()
	ts := newTestLogSpy(t)
	logger := zaptest.NewLogger(ts)
	zapLogger := zapLogger{logger: logger}

	t.Run("LifecycleOnStartEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(LifecycleOnStartEvent{Caller: "bytes.NewBuffer"})
		ts.AssertMessages("INFO\tstarting\t{\"caller\": \"bytes.NewBuffer\"}")
	})
	t.Run("LifecycleOnStopEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(LifecycleOnStopEvent{Caller: "bytes.NewBuffer"})
		ts.AssertMessages("INFO\tstopping\t{\"caller\": \"bytes.NewBuffer\"}")
	})
	t.Run("ApplyOptionsError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(ApplyOptionsError{Err: fmt.Errorf("some error")})
		ts.AssertMessages("ERROR\terror encountered while applying options\t{\"error\": \"some error\"}")
	})

	t.Run("SupplyEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(SupplyEvent{Constructor: bytes.NewBuffer})
		ts.AssertMessages("INFO\tsupplying\t{\"constructor\": \"bytes.NewBuffer()\", \"type\": \"*bytes.Buffer\"}")
	})
	t.Run("ProvideEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(ProvideEvent{bytes.NewBuffer})
		ts.AssertMessages("INFO\tproviding\t{\"constructor\": \"bytes.NewBuffer()\", \"type\": \"*bytes.Buffer\"}")
	})
	t.Run("InvokeEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(InvokeEvent{bytes.NewBuffer})
		ts.AssertMessages("INFO\tinvoke\t{\"function\": \"bytes.NewBuffer()\"}")
	})
	t.Run("InvokeFailedEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(InvokeFailedEvent{
			Function: bytes.NewBuffer,
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfx.Invoke failed\t{\"error\": \"some error\", \"stack\": \"\", \"function\": \"bytes.NewBuffer()\"}")
	})
	t.Run("StartFailureError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(StartFailureError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfailed to start\t{\"error\": \"some error\"}")
	})
	t.Run("StopSignalEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(StopSignalEvent{
			Signal: "signal",
		})
		ts.AssertMessages("INFO\treceived signal\t{\"signal\": \"SIGNAL\"}")
	})
	t.Run("StopErrorEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(StopErrorEvent{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfailed to stop cleanly\t{\"error\": \"some error\"}")
	})
	t.Run("StartRollbackError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(StartRollbackError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tcould not rollback cleanly\t{\"error\": \"some error\"}")
	})
	t.Run("StartErrorEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(StartErrorEvent{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tstartup failed, rolling back\t{\"error\": \"some error\"}")
	})
	t.Run("RunningEvent", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(RunningEvent{})
		ts.AssertMessages("INFO\trunning")
	})

}

// func TestPrint(t *testing.T) {
// 	sink := new(Spy)
//
// 	t.Run("printProvide", func(t *testing.T) {
// 		sink.Reset()
//
// 		// for _, rtype := range fxreflect.ReturnTypes(bytes.NewBuffer) {
// 			// Info("providing",
// 			// 	Field{Key: "type", Value: rtype},
// 			// 	Field{Key: "constructor", Value: fxreflect.FuncName(bytes.NewBuffer)},
// 			// ).Write(sink)
// 		sink.LogEvent(ProvideEvent{Constructor: bytes.NewBuffer})
// 		assert.Equal(t, "bytes.NewBuffer", fxreflect.FuncName(sink.Events()[0].(ProvideEvent).Constructor))
// 		// }
// 		// assert.Contains(t, sink.String(), "providing")
// 		// assert.Contains(t, sink.Fields(), zap.Field{
// 		// 	Key:    "type",
// 		// 	Type:   zapcore.StringType,
// 		// 	String: "*bytes.Buffer",
// 		// })
// 		// assert.Contains(t, sink.Fields(), zap.Field{
// 		// 	Key:    "constructor",
// 		// 	Type:   zapcore.StringType,
// 		// 	String: "bytes.NewBuffer()",
// 		// })
// 	})
//
// 	t.Run("PrintSupply", func(t *testing.T) {
// 		sink.Reset()
// 		for _, rtype := range fxreflect.ReturnTypes(func() *bytes.Buffer { return bytes.NewBuffer(nil) }) {
// 			Info("supplying", Field{Key: "type", Value: rtype}).Write(sink)
// 		}
// 		assert.Contains(t, sink.String(), "supplying")
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*bytes.Buffer",
// 		})
// 	})
//
// 	t.Run("printExpandsTypesInOut", func(t *testing.T) {
// 		sink.Reset()
//
// 		type A struct{}
// 		type B struct{}
// 		type C struct{}
// 		type Ret struct {
// 			dig.Out
// 			*A
// 			B
// 			C `name:"foo"`
// 		}
// 		for _, rtype := range fxreflect.ReturnTypes(func() Ret { return Ret{} }) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(func() Ret { return Ret{} })},
// 			).Write(sink)
// 		}
// 		assert.Contains(t, sink.String(), "providing")
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*fxlog.A",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "fxlog.B",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "fxlog.C:foo",
// 		})
// 	})
//
// 	t.Run("printHandlesDotGitCorrectly", func(t *testing.T) {
// 		sink.Reset()
// 		for _, rtype := range fxreflect.ReturnTypes(sample.New) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(sample.New)},
// 			).Write(sink)
// 		}
// 		assert.NotContains(t, sink.String(), "%2e", "should not be url encoded")
// 		assert.Contains(t, sink.String(), "providing", "should contain a dot")
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "constructor",
// 			Type:   zapcore.StringType,
// 			String: "go.uber.org/fx/internal/fxlog/sample.git.New()",
// 		})
// 	})
//
// 	t.Run("printOutNamedTypes", func(t *testing.T) {
// 		sink.Reset()
//
// 		type A struct{}
// 		type B struct{}
// 		type Ret struct {
// 			dig.Out
// 			*B `name:"foo"`
//
// 			A1 *A `name:"primary"`
// 			A2 *A `name:"secondary"`
// 		}
// 		for _, rtype := range fxreflect.ReturnTypes(func() Ret { return Ret{} }) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(func() Ret { return Ret{} })},
// 			).Write(sink)
// 		}
// 		assert.Contains(t, sink.String(), "providing")
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*fxlog.A:primary",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*fxlog.A:secondary",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*fxlog.B:foo",
// 		})
// 	})
//
// 	t.Run("printProvideInvalid", func(t *testing.T) {
// 		sink.Reset()
// 		// No logging on invalid provides, since we're already logging an error
// 		// elsewhere.
// 		for _, rtype := range fxreflect.ReturnTypes(bytes.NewBuffer(nil)) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(bytes.NewBuffer(nil))},
// 			).Write(sink)
// 		}
// 		assert.Equal(t, "", sink.String())
// 	})
//
// 	t.Run("printStripsVendorPath", func(t *testing.T) {
// 		sink.Reset()
// 		// assert is vendored within fx and is a good test case
// 		for _, rtype := range fxreflect.ReturnTypes(assert.New) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(assert.New)},
// 			).Write(sink)
// 		}
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "constructor",
// 			Type:   zapcore.StringType,
// 			String: "github.com/stretchr/testify/assert.New()",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "*assert.Assertions",
// 		})
// 	})
//
// 	t.Run("printFooVendorPath", func(t *testing.T) {
// 		sink.Reset()
// 		// assert is vendored within fx and is a good test case
// 		for _, rtype := range fxreflect.ReturnTypes(foovendor.New) {
// 			Info("providing",
// 				Field{Key: "type", Value: rtype},
// 				Field{Key: "constructor", Value: fxreflect.FuncName(foovendor.New)},
// 			).Write(sink)
// 		}
// 		assert.Contains(t, sink.String(), "providing")
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "type",
// 			Type:   zapcore.StringType,
// 			String: "string",
// 		})
// 		assert.Contains(t, sink.Fields(), zap.Field{
// 			Key:    "constructor",
// 			Type:   zapcore.StringType,
// 			String: "go.uber.org/fx/internal/fxlog/foovendor.New()",
// 		})
// 	})
//
// 	t.Run("printSignal", func(t *testing.T) {
// 		sink.Reset()
// 		sig := os.Interrupt.String()
// 		Info(sig).Write(sink)
// 		assert.Equal(t, "interrupt\n", sink.String())
// 	})
// }
