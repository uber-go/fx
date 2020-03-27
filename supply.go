// Copyright (c) 2019 Uber Technologies, Inc.
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

package fx

import (
	"reflect"
)

// Supply provides instantiated values for dependency injection as if
// they had been provided by a nullary constructor returning most specific type
// (as determined by reflection) of any given value. For example, given:
//
//  type (
// 		TypeA struct{}
//		TypeB struct{}
//	)
//  var a, b = &TypeA{}, TypeB{}
//
// The following two forms are equivalent:
//
//	fx.Provide(
//		func() *TypeA { return a },
//		func() TypeB { return b },
//	)
//
//  fx.Supply(a, b)
//
// Supply operates by devising a constructor of the first form based on
// the types of the supplied values. Supply accepts any number of arguments,
// but with the following caveats:
//
// (1) Supply panics when given a naked nil, or an error value.
// (2) When given a fx.Annotated, the target is expected to be a value.
//     Supply replaces the target with a constructor function.
//
func Supply(values ...interface{}) Option {
	if len(values) == 0 {
		return Options()
	}

	returnTypes := make([]reflect.Type, len(values))
	returnValues := make([]reflect.Value, len(values))

	for i, value := range values {
		returnTypes[i] = reflect.TypeOf(value)
		returnValues[i] = reflect.ValueOf(value)
	}

	ft := reflect.FuncOf([]reflect.Type{}, returnTypes, false)
	fv := reflect.MakeFunc(ft, func([]reflect.Value) []reflect.Value {
		return returnValues
	})

	return Provide(fv.Interface())
}
