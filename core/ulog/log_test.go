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
	"net"
	"testing"
	"time"

	"go.uber.org/fx/core/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/zap"
)

func TestSimpleLogger(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := NewBuilder().SetLogger(zaplogger).Build()

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
		log := NewBuilder().SetLogger(zaplogger).Build().With("method", "test_method")
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
		log := NewBuilder().SetLogger(zaplogger).Build()
		log.Info("info message", "c")
		log.Info("info message", "c", "d", "e")
		log.DFatal("debug message")
		assert.Equal(t, []string{
			`{"level":"info","msg":"info message","error":"invalid number of arguments"}`,
			`{"level":"info","msg":"info message","error":"invalid number of arguments"}`,
			`{"level":"error","msg":"debug message"}`,
		}, buf.Lines(), "Incorrect output from logger")
	})
}

func TestFatalsAndPanics(t *testing.T) {
	testutils.WithInMemoryLogger(t, nil, func(zaplogger zap.Logger, buf *testutils.TestBuffer) {
		log := NewBuilder().SetLogger(zaplogger).Build()
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
	log := NewBuilder().Build()
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
	log := NewBuilder().Build()
	assert.NotNil(t, log.RawLogger())
}

func TestLogger(t *testing.T) {
	log := Logger()
	assert.NotNil(t, log.RawLogger())
}
