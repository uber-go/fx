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

package fx

import (
	"fmt"
	"reflect"
)

// Inject fills the given struct with values from the dependency injection
// container on application start.
//
// The target MUST be a pointer to a struct. Only exported fields will be
// filled.
func Inject(target interface{}) Option {
	v := reflect.ValueOf(target)

	if t := v.Type(); t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return Invoke(func() error {
			return fmt.Errorf("Inject expected a pointer to a struct, got a %v", t)
		})
	}

	v = v.Elem()
	t := v.Type()

	// We generate a function with one argument for each field in the target
	// struct.

	argTypes := make([]reflect.Type, 0, t.NumField())

	// List of values in the target struct aligned with the arguments of the
	// generated function.
	//
	// So for example, if the target is,
	//
	// 	var target struct {
	// 		Foo io.Reader
	// 		bar []byte
	// 		Baz io.Writer
	// 	}
	//
	// The generated function has the shape,
	//
	// 	func(io.Reader, io.Writer)
	//
	// And `targets` is,
	//
	// 	[
	// 		target.Field(0),  // Foo io.Reader
	// 		target.Field(2),  // Baz io.Writer
	// 	]
	//
	// As we iterate through the arguments received by the function, we can
	// simply copy the value into the corresponding value in the targets list.
	targets := make([]reflect.Value, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// Skip private fields.
		if f.PkgPath != "" {
			continue
		}

		argTypes = append(argTypes, f.Type)
		targets = append(targets, v.Field(i))
	}

	// Equivalent to,
	//
	// 	func(foo Foo, bar Bar) {
	// 		target.Foo = foo
	// 		target.Bar = bar
	// 	}

	fn := reflect.MakeFunc(
		reflect.FuncOf(argTypes, nil /* results */, false /* variadic */),
		func(args []reflect.Value) []reflect.Value {
			for i, arg := range args {
				targets[i].Set(arg)
			}
			return nil
		},
	)

	return Invoke(fn.Interface())
}
