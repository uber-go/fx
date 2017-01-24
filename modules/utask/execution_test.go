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

package utask

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	_backend = &InMemBackend{}
}

func TestEnqueueNonFunction(t *testing.T) {
	err := Register("I am not a function")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "a func as input but was")
}

func TestEnqueueWithMultipleReturnValues(t *testing.T) {
	fn := func() (string, error) { return "", nil }
	err := Register(fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "return only error but found")
}

func TestEnqueueFnDoesNotReturnError(t *testing.T) {
	fn := func() string { return "" }
	err := Register(fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "return error but found")
}

func TestEnqueueFnWithMismatchedArgCount(t *testing.T) {
	fn := func(s string) error { return nil }
	err := Register(fn)
	assert.NoError(t, err)
	err = Enqueue(fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "1 function arg(s) but found 0")
}

func TestEnqueueFnWithMismatchedArgType(t *testing.T) {
	fn := func(s string) error { return nil }
	err := Register(fn)
	assert.NoError(t, err)
	err = Enqueue(fn, 1)
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "argument: 1 from type: int to type: string",
	)
}

func TestEnqueueAndConsumeWithoutRegister(t *testing.T) {
	fn := func(num float64) error { return nil }
	err := Enqueue(fn, float64(1.0))
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "\"go.uber.org/fx/modules/utask.TestEnqueueAndConsumeWithoutRegister.func1\""+
			" not found",
	)
}

func TestEnqueueEncodingError(t *testing.T) {
	fn := func(car Car) error { return nil }
	fnLookup.fnNameMap[getFunctionName(fn)] = fn
	err := Enqueue(fn, Car{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to encode the function")
}

func TestRunDecodeError(t *testing.T) {
	err := Run([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to decode the message")
}

func TestEnqueueNoArgsFn(t *testing.T) {
	err := Register(NoArgs)
	assert.NoError(t, err)
	err = Enqueue(NoArgs)
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.NoError(t, err)
}

func TestEnqueueSimpleFn(t *testing.T) {
	err := Register(SimpleWithError)
	assert.NoError(t, err)
	err = Enqueue(SimpleWithError, "hello")
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Simple error")
}

func TestEnqueueMapFn(t *testing.T) {
	fn := func(map[string]string) error { return nil }
	err := Register(fn)
	assert.NoError(t, err)
	err = Enqueue(fn, make(map[string]string))
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.NoError(t, err)
}

func TestEnqueueComplexFnWithError(t *testing.T) {
	err := Register(Complex)
	assert.NoError(t, err)
	err = Enqueue(Complex, Car{Brand: "infinity", Year: 2017})
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Complex error")
	err = Enqueue(Complex, Car{Brand: "honda", Year: 2017})
	assert.NoError(t, err)
	err = GlobalBackend().Consume()
	assert.NoError(t, err)
}

func NoArgs() error {
	return nil
}

func SimpleWithError(a string) error {
	return fmt.Errorf("Simple error")
}

type Car struct {
	Brand string
	Year  int
}

func Complex(car Car) error {
	if car.Brand == "infinity" {
		return fmt.Errorf("Complex error")
	}
	return nil
}
