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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxlog/foovendor"
	"go.uber.org/fx/internal/fxlog/sample.git"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() {
		DefaultLogger(testutil.WriteSyncer{T: t})
	})
}

func TestPrint(t *testing.T) {
	sink := new(Spy)

	t.Run("printf", func(t *testing.T) {
		sink.Reset()
		Info("foo 42").Write(sink)
		assert.Equal(t, "foo 42\n", sink.String())
	})

	t.Run("printProvide", func(t *testing.T) {
		sink.Reset()
		for _, rtype := range fxreflect.ReturnTypes(bytes.NewBuffer) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(bytes.NewBuffer)},
			).Write(sink)
		}
		assert.Contains(t, sink.String(), "providing")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*bytes.Buffer",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "constructor",
			Type:   zapcore.StringType,
			String: "bytes.NewBuffer()",
		})
	})

	t.Run("PrintSupply", func(t *testing.T) {
		sink.Reset()
		for _, rtype := range fxreflect.ReturnTypes(func() *bytes.Buffer { return bytes.NewBuffer(nil) }) {
			Info("supplying", Field{Key: "type", Value: rtype}).Write(sink)
		}
		assert.Contains(t, sink.String(), "supplying")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*bytes.Buffer",
		})
	})

	t.Run("printExpandsTypesInOut", func(t *testing.T) {
		sink.Reset()

		type A struct{}
		type B struct{}
		type C struct{}
		type Ret struct {
			dig.Out
			*A
			B
			C `name:"foo"`
		}
		for _, rtype := range fxreflect.ReturnTypes(func() Ret { return Ret{} }) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(func() Ret { return Ret{} })},
			).Write(sink)
		}
		assert.Contains(t, sink.String(), "providing")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*fxlog.A",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "fxlog.B",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "fxlog.C:foo",
		})
	})

	t.Run("printHandlesDotGitCorrectly", func(t *testing.T) {
		sink.Reset()
		for _, rtype := range fxreflect.ReturnTypes(sample.New) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(sample.New)},
			).Write(sink)
		}
		assert.NotContains(t, sink.String(), "%2e", "should not be url encoded")
		assert.Contains(t, sink.String(), "providing", "should contain a dot")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "constructor",
			Type:   zapcore.StringType,
			String: "go.uber.org/fx/internal/fxlog/sample.git.New()",
		})
	})

	t.Run("printOutNamedTypes", func(t *testing.T) {
		sink.Reset()

		type A struct{}
		type B struct{}
		type Ret struct {
			dig.Out
			*B `name:"foo"`

			A1 *A `name:"primary"`
			A2 *A `name:"secondary"`
		}
		for _, rtype := range fxreflect.ReturnTypes(func() Ret { return Ret{} }) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(func() Ret { return Ret{} })},
			).Write(sink)
		}
		assert.Contains(t, sink.String(), "providing")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*fxlog.A:primary",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*fxlog.A:secondary",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*fxlog.B:foo",
		})
	})

	t.Run("printProvideInvalid", func(t *testing.T) {
		sink.Reset()
		// No logging on invalid provides, since we're already logging an error
		// elsewhere.
		for _, rtype := range fxreflect.ReturnTypes(bytes.NewBuffer(nil)) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(bytes.NewBuffer(nil))},
			).Write(sink)
		}
		assert.Equal(t, "", sink.String())
	})

	t.Run("printStripsVendorPath", func(t *testing.T) {
		sink.Reset()
		// assert is vendored within fx and is a good test case
		for _, rtype := range fxreflect.ReturnTypes(assert.New) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(assert.New)},
			).Write(sink)
		}
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "constructor",
			Type:   zapcore.StringType,
			String: "github.com/stretchr/testify/assert.New()",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "*assert.Assertions",
		})
	})

	t.Run("printFooVendorPath", func(t *testing.T) {
		sink.Reset()
		// assert is vendored within fx and is a good test case
		for _, rtype := range fxreflect.ReturnTypes(foovendor.New) {
			Info("providing",
				Field{Key: "type", Value: rtype},
				Field{Key: "constructor", Value: fxreflect.FuncName(foovendor.New)},
			).Write(sink)
		}
		assert.Contains(t, sink.String(), "providing")
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "type",
			Type:   zapcore.StringType,
			String: "string",
		})
		assert.Contains(t, sink.Fields(), zap.Field{
			Key:    "constructor",
			Type:   zapcore.StringType,
			String: "go.uber.org/fx/internal/fxlog/foovendor.New()",
		})
	})

	t.Run("printSignal", func(t *testing.T) {
		sink.Reset()
		sig := os.Interrupt.String()
		Info(sig).Write(sink)
		assert.Equal(t, "interrupt\n", sink.String())
	})
}
