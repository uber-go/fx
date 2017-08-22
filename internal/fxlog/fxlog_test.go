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

package fxlog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxlog/foovendor"
	sample "go.uber.org/fx/internal/fxlog/sample.git"
)

type spy struct {
	*bytes.Buffer
}

func newSpy() *spy {
	return &spy{bytes.NewBuffer(nil)}
}

func (s *spy) Printf(format string, is ...interface{}) {
	fmt.Fprintln(s, fmt.Sprintf(format, is...))
}

// stubs the exit call, returns a function that restores a real exit function
// and asserts that the stub was called.
func stubExit() func(testing.TB) {
	prev := _exit
	var called bool
	_exit = func() { called = true }
	return func(t testing.TB) {
		assert.True(t, called, "Exit wasn't called.")
		_exit = prev
	}
}

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() { New() })
}

func TestPrint(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}

	t.Run("printf", func(t *testing.T) {
		sink.Reset()
		logger.Printf("foo %d", 42)
		assert.Equal(t, "[Fx] foo 42\n", sink.String())
	})

	t.Run("printProvide", func(t *testing.T) {
		sink.Reset()
		logger.PrintProvide(bytes.NewBuffer)
		assert.Equal(t, "[Fx] PROVIDE\t*bytes.Buffer <= bytes.NewBuffer()\n", sink.String())
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
		logger.PrintProvide(func() Ret { return Ret{} })

		s := sink.String()
		assert.Contains(t, s, "[Fx] PROVIDE\t*fxlog.A <=")
		assert.Contains(t, s, "[Fx] PROVIDE\tfxlog.B <=")
		assert.Contains(t, s, "[Fx] PROVIDE\tfxlog.C:foo <=")
	})

	t.Run("printHandlesDotGitCorrectly", func(t *testing.T) {
		sink.Reset()
		logger.PrintProvide(sample.New)
		assert.NotContains(t, sink.String(), "%2e", "should not be url encoded")
		assert.Contains(t, sink.String(), "sample.git", "should contain a dot")
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
		logger.PrintProvide(func() Ret { return Ret{} })

		s := sink.String()
		assert.Contains(t, s, "[Fx] PROVIDE\t*fxlog.A:primary <=")
		assert.Contains(t, s, "[Fx] PROVIDE\t*fxlog.A:secondary <=")
		assert.Contains(t, s, "[Fx] PROVIDE\t*fxlog.B:foo <=")
	})

	t.Run("printProvideInvalid", func(t *testing.T) {
		sink.Reset()
		// No logging on invalid provides, since we're already logging an error
		// elsewhere.
		logger.PrintProvide(bytes.NewBuffer(nil))
		assert.Equal(t, "", sink.String())
	})

	t.Run("printStripsVendorPath", func(t *testing.T) {
		sink.Reset()
		// assert is vendored within fx and is a good test case
		logger.PrintProvide(assert.New)
		assert.Contains(
			t, sink.String(),
			"*assert.Assertions <= vendor/github.com/stretchr/testify/assert.New()")
	})

	t.Run("printFooVendorPath", func(t *testing.T) {
		sink.Reset()
		// assert is vendored within fx and is a good test case
		logger.PrintProvide(foovendor.New)
		assert.Contains(
			t, sink.String(),
			"string <= go.uber.org/fx/internal/fxlog/foovendor.New()")
	})

	t.Run("printSignal", func(t *testing.T) {
		sink.Reset()
		logger.PrintSignal(os.Interrupt)
		assert.Equal(t, "[Fx] INTERRUPT\n", sink.String())
	})
}

func TestPanic(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}
	assert.Panics(t, func() { logger.Panic(errors.New("foo")) })
	assert.Equal(t, "[Fx] foo\n", sink.String())
}

func TestFatal(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}

	undo := stubExit()
	defer undo(t)

	logger.Fatalf("foo %d", 42)
	assert.Equal(t, "[Fx] foo 42\n", sink.String())
}
