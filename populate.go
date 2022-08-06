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
	"fmt"
	"reflect"
)

// Populate sets targets with values from the dependency injection container
// during application initialization. All targets must be pointers to the
// values that must be populated. Pointers to structs that embed In are
// supported, which can be used to populate multiple values in a struct.
//
// This is most helpful in unit tests: it lets tests leverage Fx's automatic
// constructor wiring to build a few structs, but then extract those structs
// for further testing.
func Populate(targets ...interface{}) Option {
	// Fields of the generated fx.In struct.
	fields := make([]reflect.StructField, 0, len(targets)+1)

	// Anonymous dig.In field.
	fields = append(fields, reflect.StructField{
		Name:      _typeOfIn.Name(),
		Anonymous: true,
		Type:      _typeOfIn,
	})

	// Build struct that looks like:
	//
	// struct {
	//   fx.In
	//
	//   F0 SomeType
	//   F1 SomeType
	//   [...]
	// }
	targetsValue := make([]reflect.Value, len(targets))
	for i, t := range targets {
		if t == nil {
			return Error(fmt.Errorf("failed to Populate: target %v is nil", i+1))
		}

		// support fx.Annotated
		if anno, ok := t.(Annotated); ok {
			if reflect.TypeOf(anno.Target).Kind() != reflect.Ptr {
				return Error(fmt.Errorf("failed to Populate: target %v is not a pointer type, got %T", i+1, anno.Target))
			}
			targetsValue[i] = reflect.ValueOf(anno.Target).Elem()

			var tag string
			if anno.Name != "" {
				tag = fmt.Sprintf(`name:"%s"`, anno.Name)
			} else if anno.Group != "" {
				tag = fmt.Sprintf(`group:"%s"`, anno.Group)
			}

			fields = append(fields, reflect.StructField{
				Name: fmt.Sprintf("F%d", i),
				Type: targetsValue[i].Type(),
				Tag:  reflect.StructTag(tag),
			})
			continue
		}

		rt := reflect.TypeOf(t)
		if rt.Kind() != reflect.Ptr {
			return Error(fmt.Errorf("failed to Populate: target %v is not a pointer type, got %T", i+1, t))
		}

		targetsValue[i] = reflect.ValueOf(t).Elem()
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("F%d", i),
			Type: targetsValue[i].Type(),
		})
	}

	// Build a function that looks like:
	//
	// func(in inType) {
	//   *targets[0] = in.T0
	//   *targets[1] = in.T1
	//   [...]
	// }
	fn := reflect.MakeFunc(
		reflect.FuncOf(
			[]reflect.Type{reflect.StructOf(fields)},
			nil,   /* results */
			false, /* variadic */
		),
		func(args []reflect.Value) []reflect.Value {
			result := args[0]
			for i := 1; i < result.NumField(); i++ {
				targetsValue[i-1].Set(result.Field(i))
			}
			return nil
		},
	)

	return Invoke(fn.Interface())
}
