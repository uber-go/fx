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
	"context"
	"fmt"
	"reflect"
)

var _ctxType = typeOfInterface(new(context.Context))
var _lifecycleType = typeOfInterface(new(Lifecycle))
var _errType = typeOfInterface(new(error))

func typeOfInterface(newIfacePtr interface{}) reflect.Type {
	return reflect.TypeOf(newIfacePtr).Elem()
}

// OnStart registers functions that are executed when the application is started.
// Arguments for these invocations are built using the constructors provided by
// Provide. Passing multiple OnStart options appends new functions to the applications'
// existing list.
//
// Similar to Invokes, functions are always executed, and they are run in order.
// OnStart functions may return no values, or an error. Any other return values
// cause errors.
//
// If the first argument of the passed in function is a context.Context, then the
// context from Start is passed in.
func OnStart(funcs ...interface{}) Option {
	var opts []Option
	for _, f := range funcs {
		opts = append(opts, onStart(reflect.ValueOf(f)))
	}
	return Options(opts...)
}

func validateOnStartFunc(ft reflect.Type) error {
	if kind := ft.Kind(); kind != reflect.Func {
		return fmt.Errorf("OnStart must be passed a function, got %v", kind)
	}
	switch ft.NumOut() {
	case 0:
		// No return value, nothing to validate.
	case 1:
		// Must be error
		if ft.Out(0) != _errType {
			return fmt.Errorf("OnStart functions can only return an error, found %v", ft.Out(0))
		}
	default:
		return fmt.Errorf("OnStart functions must have no returns, or a single error return")
	}

	return nil
}

func onStart(f reflect.Value) Option {
	ft := f.Type()

	if err := validateOnStartFunc(ft); err != nil {
		return invokeErr(err)
	}

	// If the first argument is a context, ignore it, but track we need it for later.
	isFirstCtx := ft.NumIn() > 0 && ft.In(0) == _ctxType

	// The invoke function will take a lifecycle, followed by any arguments to f.
	invokeInTypes := []reflect.Type{_lifecycleType}
	for i := 0; i < ft.NumIn(); i++ {
		if i == 0 && isFirstCtx {
			continue
		}

		invokeInTypes = append(invokeInTypes, ft.In(i))
	}

	invokeFuncType := reflect.FuncOf(invokeInTypes, []reflect.Type{_errType}, false /* variadic */)

	invokeFunc := reflect.MakeFunc(invokeFuncType, func(args []reflect.Value) []reflect.Value {
		// The first argument to invoke is a lifecycle, let's extract that.
		lifecycle := args[0].Interface().(Lifecycle)
		lifecycle.Append(Hook{
			OnStart: func(ctx context.Context) error {
				var startArgs []reflect.Value

				if isFirstCtx {
					startArgs = append(startArgs, reflect.ValueOf(ctx))
				}

				startArgs = append(startArgs, args[1:]...)
				f.Call(startArgs)
				return nil
			},
		})

		return []reflect.Value{reflect.New(_errType).Elem()}
	})

	return Invoke(invokeFunc.Interface())
}
