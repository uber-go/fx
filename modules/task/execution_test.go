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

package task

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _errorCh <-chan error

func init() {
	_globalBackend = NewInMemBackend()
	_errorCh = _globalBackend.Start(make(chan struct{}))
}

func TestRegisterNonFunction(t *testing.T) {
	err := Register("I am not a function")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a func as input but was")
}

func TestRegisterWithMultipleReturnValues(t *testing.T) {
	fn := func() (string, error) { return "", nil }
	err := Register(fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "return only error but found")
}

func TestRegisterFnDoesNotReturnError(t *testing.T) {
	fn := func() string { return "" }
	err := Register(fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "return error but found")
}

func TestRegisterFnWithMismatchedArgCount(t *testing.T) {
	fn := func(s string) error { return nil }
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 function arg(s) but found 0")
}

func TestEnqueueFnWithMismatchedArgType(t *testing.T) {
	fn := func(s string) error { return nil }
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn, 1)
	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "argument: 1 from type: int to type: string",
	)
}

func TestEnqueueWithoutRegister(t *testing.T) {
	fn := func(num float64) error { return nil }
	err := Enqueue(fn, float64(1.0))
	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "\"go.uber.org/fx/modules/task.TestEnqueueWithoutRegister.func1\""+
			" not found",
	)
}

func TestConsumeWithoutRegister(t *testing.T) {
	fn := func(num float64) error { return nil }
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn, float64(1.0))
	require.NoError(t, err)
	fnLookup.setFnNameMap(make(map[string]interface{}))
	err = <-_errorCh
	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "\"go.uber.org/fx/modules/task.TestConsumeWithoutRegister.func1\""+
			" not found",
	)
}

func TestEnqueueEncodingError(t *testing.T) {
	fn := func(car Car) error { return nil }
	fnLookup.addFn(getFunctionName(fn), fn)
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn, Car{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to encode the function")
}

func TestRunDecodeError(t *testing.T) {
	err := Run([]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to decode the message")
}

func TestEnqueueNoArgsFn(t *testing.T) {
	err := Register(NoArgs)
	require.NoError(t, err)
	err = Enqueue(NoArgs)
	require.NoError(t, err)
	err = <-_errorCh
	require.NoError(t, err)
}

func TestEnqueueSimpleFn(t *testing.T) {
	err := Register(SimpleWithError)
	require.NoError(t, err)
	err = Enqueue(SimpleWithError, "hello")
	require.NoError(t, err)
	err = <-_errorCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Simple error")
}

func TestEnqueueMapFn(t *testing.T) {
	fn := func(map[string]string) error { return nil }
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn, make(map[string]string))
	require.NoError(t, err)
	err = <-_errorCh
	require.NoError(t, err)
}

func TestEnqueueFnClosure(t *testing.T) {
	var wg sync.WaitGroup
	fn := func() error { return nil }
	wg.Add(1)
	go func() {
		i := 1
		defer wg.Done()
		fn = func() error {
			i = i + 1
			if i == 2 {
				return nil
			}
			return errors.New("Unexpected i")
		}
	}()
	wg.Wait()
	err := Register(fn)
	require.NoError(t, err)
	err = Enqueue(fn)
	require.NoError(t, err)
	err = <-_errorCh
	require.NoError(t, err)
}

func TestEnqueueComplexFnWithError(t *testing.T) {
	err := Register(Complex)
	require.NoError(t, err)
	err = Enqueue(Complex, Car{Brand: "infinity", Year: 2017})
	require.NoError(t, err)
	err = <-_errorCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Complex error")
	err = Enqueue(Complex, Car{Brand: "honda", Year: 2017})
	require.NoError(t, err)
	err = <-_errorCh
	require.NoError(t, err)
}

func NoArgs() error {
	return nil
}

func SimpleWithError(a string) error {
	return errors.New("Simple error")
}

type Car struct {
	Brand string
	Year  int
}

func Complex(car Car) error {
	if car.Brand == "infinity" {
		return errors.New("Complex error")
	}
	return nil
}

func TestCastToError(t *testing.T) {
	s := make(map[string]string)
	err := castToError(reflect.ValueOf(s))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "be error but found")
}
