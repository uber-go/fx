// Copyright (c) 2022 Uber Technologies, Inc.
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
	"strings"

	"go.uber.org/fx/internal/fxreflect"
)

// Replace provides instantiated values for graph modification. Similar to
// what fx.Supply is to fx.Provide, values provided by fx.Replace behaves
// similarly to values produced by decorators specified with fx.Decorate.
//
// Refer to the documentation on fx.Decorate to see how graph modifications
// work with fx.Module.
//
// Replace panics if a value (or annotation target) is an untyped nil or an error.
func Replace(values ...interface{}) Option {
	decorators := make([]interface{}, len(values)) // one function per value
	types := make([]reflect.Type, len(values))
	for i, value := range values {
		switch value := value.(type) {
		case annotated:
			var typ reflect.Type
			value.Target, typ = newReplaceDecorator(value.Target)
			decorators[i] = value
			types[i] = typ
		default:
			decorators[i], types[i] = newReplaceDecorator(value)
		}
	}

	return replaceOption{
		Targets: decorators,
		Types:   types,
		Stack:   fxreflect.CallerStack(1, 0),
	}
}

type replaceOption struct {
	Targets []interface{}
	Types   []reflect.Type // type of value produced by constructor[i]
	Stack   fxreflect.Stack
}

func (o replaceOption) apply(m *module) {
	for _, target := range o.Targets {
		m.decorators = append(m.decorators, decorator{
			Target: target,
			Stack:  o.Stack,
		})
	}
}

func (o replaceOption) String() string {
	items := make([]string, 0, len(o.Targets))
	for _, typ := range o.Types {
		items = append(items, typ.String())
	}
	return fmt.Sprintf("fx.Replace(%s)", strings.Join(items, ", "))
}

// Returns a function that takes no parameters, and returns the given value.
func newReplaceDecorator(value interface{}) (interface{}, reflect.Type) {
	switch value.(type) {
	case nil:
		panic("untyped nil passed to fx.Replace")
	case error:
		panic("error value passed to fx.Replace")
	}

	typ := reflect.TypeOf(value)
	returnTypes := []reflect.Type{typ}
	returnValues := []reflect.Value{reflect.ValueOf(value)}

	ft := reflect.FuncOf([]reflect.Type{}, returnTypes, false)
	fv := reflect.MakeFunc(ft, func([]reflect.Value) []reflect.Value {
		return returnValues
	})

	return fv.Interface(), typ
}
