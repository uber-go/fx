package fx

import (
	"reflect"
)

// Supply makes the given values available for dependency resolution as if
// they had been provided by a constructor. Specifically, given:
//
//  type MyStruct struct{}
//  var myStruct = &MyStruct{}
//
// The following two forms are equivalent:
//
//  fx.Provide(func() *MyStruct { return myStruct })
//  fx.Supply(myStruct)
//
// Supply operates by constructing a constructor of the first form based on
// the types of the supplied values. Supply accepts any number of arguments.
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
