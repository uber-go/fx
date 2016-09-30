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

package util

import (
	"reflect"
)

func FindField(instance interface{}, name *string, rt reflect.Type) (reflect.Value, bool) {
	// walk the fields looking for ones of type Service
	//
	objType := reflect.TypeOf(instance)

	for objType.Kind() == reflect.Ptr || objType.Kind() == reflect.Interface {
		objType = objType.Elem()
	}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.FieldByIndex([]int{i})

		if name != nil && field.Name != *name {
			continue
		}

		if rt != nil && !field.Type.AssignableTo(rt) {
			continue
		}
		// if we got this far, return the value.
		//
		val := reflect.Indirect(reflect.ValueOf(instance))
		return val.FieldByIndex(field.Index), true
	}
	return reflect.Value{}, false
}
