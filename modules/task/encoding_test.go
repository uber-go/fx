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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type encodingTest struct {
	encoding       Encoding
	inputObj       interface{}
	verifyEncoding bool
}

var kvMap = map[string]string{"key": "value"}

var encodingTests = []encodingTest{
	{&NopEncoding{}, kvMap, false},
	{&GobEncoding{}, kvMap, true},
}

func TestEncoding(t *testing.T) {
	for _, test := range encodingTests {
		testEncMethods(t, test.encoding, test.inputObj, test.verifyEncoding)
	}
}

func testEncMethods(t *testing.T, encoding Encoding, obj interface{}, deepChecks bool) {
	assert.NoError(t, encoding.Register(obj))
	msg, err := encoding.Marshal(obj)
	assert.NoError(t, err)
	receivedObj := make(map[string]string)
	assert.NoError(t, encoding.Unmarshal(msg, &receivedObj))
	if deepChecks {
		assert.True(t, reflect.DeepEqual(obj, receivedObj))
	}
}
