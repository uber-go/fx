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
// Annotating each pointer with ParamTags is also supported as a shorthand
// to passing a pointer to a struct that embeds In with field tags. For example:
//
//	 var a A
//	 var b B
//	 fx.Populate(
//		fx.Annotate(
//				&a,
//				fx.ParamTags(`name:"A"`)
//	 	),
//		fx.Annotate(
//				&b,
//				fx.ParamTags(`name:"B"`)
//	 	)
//	 )
//
// Code above is equivalent to the following:
//
//	type Target struct {
//		fx.In
//
//		a A `name:"A"`
//		b B `name:"B"`
//	}
//	var target Target
//	...
//	fx.Populate(&target)
//
// This is most helpful in unit tests: it lets tests leverage Fx's automatic
// constructor wiring to build a few structs, but then extract those structs
// for further testing.
func Populate(targets ...interface{}) Option {
	// Validate all targets are non-nil pointers.
	fields := make([]reflect.StructField, len(targets)+1)
	fields[0] = reflect.StructField{
		Name:      "In",
		Type:      reflect.TypeOf(In{}),
		Anonymous: true,
	}
	for i, t := range targets {
		if t == nil {
			return Error(fmt.Errorf("failed to Populate: target %v is nil", i+1))
		}
		var (
			rt  reflect.Type
			tag reflect.StructTag
		)
		switch t := t.(type) {
		case annotated:
			rt = reflect.TypeOf(t.Target)
			tag = reflect.StructTag(t.ParamTags[0])
			targets[i] = t.Target
		default:
			rt = reflect.TypeOf(t)
		}
		if rt.Kind() != reflect.Ptr {
			return Error(fmt.Errorf("failed to Populate: target %v is not a pointer type, got %T", i+1, t))
		}
		fields[i+1] = reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: rt.Elem(),
			Tag:  tag,
		}
	}

	// Build a function that looks like:
	//
	// func(t1 T1, t2 T2, ...) {
	//   *targets[0] = t1
	//   *targets[1] = t2
	//   [...]
	// }
	//
	fnType := reflect.FuncOf([]reflect.Type{reflect.StructOf(fields)}, nil, false /* variadic */)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		arg := args[0]
		for i, target := range targets {
			reflect.ValueOf(target).Elem().Set(arg.Field(i + 1))
		}
		return nil
	})
	return Invoke(fn.Interface())
}
