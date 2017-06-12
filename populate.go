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

	"go.uber.org/dig"
)

var _digInType = reflect.TypeOf(dig.In{})

// Populate fills the given struct with values from the DI container when
// passed to App.Start.
//
// 	var target struct {
// 		Dispatcher *yarpc.Dispatcher
// 	}
// 	err := app.Start(ctx, Populate(&target))
//
// The target MUST be a pointer to a struct. Only exported fields will be
// filled. All field tags supported by dig.In are supported on Populate
// targets.
func Populate(target interface{}) interface{} {
	v := reflect.ValueOf(target)

	if t := v.Type(); t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected a pointer to a struct, got a %v", t))
	}

	v = v.Elem()
	t := v.Type()

	// We generate a struct with the same fields as the target and an embedded
	// dig.In field.

	fields := make([]reflect.StructField, 0, t.NumField()+1)

	// The fix for https://github.com/golang/go/issues/18780 requires that
	// StructField.Name is always set but older versions of Go expect Name to
	// be empty for embedded fields.
	//
	// We use populate_go19 and populate_pre_go19 with build tags to support
	// both behaviors.
	fields = append(fields, digField())

	// List of values in the target struct aligned with the fields of the
	// generated struct.
	//
	// So for example, if the target is,
	//
	// 	var target struct {
	// 		Foo io.Reader
	// 		bar []byte
	// 		Baz io.Writer
	// 	}
	//
	// The generated struct type is,
	//
	// 	struct {
	// 		dig.In
	// 		Foo io.Reader
	// 		Baz io.Writer
	// 	}
	//
	// And `targets` is,
	//
	// 	[
	// 		target.Field(0),  // Foo io.Reader
	// 		target.Field(2),  // Baz io.Writer
	// 	]
	//
	// As we iterate through the fields of the generated structs (after
	// skipping dig.In), we can simply copy the field into the corresponding
	// value in the targets list.
	targets := make([]reflect.Value, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Skip private fields.
		if f.PkgPath != "" {
			continue
		}

		if f.Type == _digInType && f.Anonymous {
			// If the struct has dig.In already embedded, skip.
			continue
		}

		fields = append(fields, reflect.StructField{
			Name:      f.Name,
			Type:      f.Type,
			Tag:       f.Tag,
			Anonymous: f.Anonymous,
		})
		targets = append(targets, v.Field(i))
	}

	// Equivalent to,
	//
	// 	func(args struct {
	// 		dig.In
	// 		Foo Foo
	// 		Bar Bar
	// 	}) {
	// 		target.Foo = args.Foo
	// 		target.Bar = args.Bar
	// 	}

	fn := reflect.MakeFunc(
		reflect.FuncOf(
			[]reflect.Type{reflect.StructOf(fields)},
			nil,   /* results */
			false, /* variadic */
		),
		func(args []reflect.Value) []reflect.Value {
			got := args[0]
			for i := 1; i < got.NumField(); i++ {
				targets[i-1].Set(got.Field(i))
			}
			return nil
		},
	)

	return fn.Interface()
}
