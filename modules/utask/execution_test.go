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

package utask

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleFn(t *testing.T) {
	err := Register(Simple, "hello")
	assert.NoError(t, err)
	err = RunNext()
	assert.NoError(t, err)
}

func TestComplexFn(t *testing.T) {
	err := Register(Complex, Car{brand: "infinity", model: "g37", year: 2017})
	assert.NoError(t, err)
	err = RunNext()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Complex error")
}

func Simple(a string) error {
	fmt.Printf("Inside function: %s\n", a)
	return nil
}

type Car struct {
	brand string
	model string
	year  int
}

func Complex(car Car) error {
	fmt.Printf("Car: %v\n", car)
	return fmt.Errorf("Complex error")
}
