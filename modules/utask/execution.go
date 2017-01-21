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

package utask

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"runtime"

	"go.uber.org/fx/ulog"
)

// Signature represents a function and its arguments
type Signature struct {
	FnName string
	Args   []interface{}
}

// Execute executes the function
func (s *Signature) Execute() ([]reflect.Value, error) {
	var targetArgs []reflect.Value
	for _, arg := range s.Args {
		targetArgs = append(targetArgs, reflect.ValueOf(arg))
	}
	if fn, ok := fnNameMap[s.FnName]; ok {
		fnValue := reflect.ValueOf(fn)
		return fnValue.Call(targetArgs), nil
	}
	return nil, fmt.Errorf("Function: %s not found. Did you forget to register?", s.FnName)
}

var bufQueue [][]byte
var fnNameMap = make(map[string]interface{})

// Enqueue sends a func before sending to the task queue
func Enqueue(fn interface{}, args ...interface{}) error {
	fnType := reflect.TypeOf(fn)
	if err := validateAsyncFn(fnType); err != nil {
		return err
	}
	if fnType.NumIn() != len(args) {
		return fmt.Errorf("Expected %d function args but found %d\n", fnType.NumIn(), len(args))
	}
	var argValues []reflect.Value
	for i := 0; i < fnType.NumIn(); i++ {
		arg := reflect.ValueOf(args[i])
		argType := reflect.TypeOf(args[i])
		if !argType.AssignableTo(fnType.In(i)) {
			return fmt.Errorf(
				"Cannot assign function argument: %d from type: %s to type: %s\n",
				i, argType, fnType.In(i),
			)
		}
		fmt.Println("Register", argType, arg)
		gob.Register(arg)
		argValues = append(argValues, arg)
	}
	fnName := getFunctionName(fn)
	fnNameMap[fnName] = fn
	s := Signature{FnName: fnName, Args: args}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		ulog.Logger().Error("Encode error:", "error", err)
		return err
	}
	sBytes := buf.Bytes()
	bufQueue = append(bufQueue, sBytes)
	return GlobalBackend().Publish(sBytes, nil)
}

// RunNextByte runs next function from queue
func RunNextByte() error {
	if len(bufQueue) > 0 {
		dec := gob.NewDecoder(bytes.NewBuffer(bufQueue[0]))
		bufQueue = bufQueue[1:]
		var s Signature
		if err := dec.Decode(&s); err != nil {
			ulog.Logger().Error("Decode error:", err)
			return err
		}
		retValues, err := s.Execute()
		if err != nil {
			return err
		}
		return castToError(retValues[0])
	}
	return nil
}

// validateAsyncFn verifies that the type is a function type that returns only an error
func validateAsyncFn(fnType reflect.Type) error {
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("Expected a func as input but was %s\n", fnType.Kind())
	}
	if fnType.NumOut() != 1 {
		return fmt.Errorf(
			"Expected function to return only error but found %d return values\n", fnType.NumOut(),
		)
	}
	if !isError(fnType.Out(0)) {
		return fmt.Errorf(
			"Expected function to return error but found %d\n", fnType.Out(0).Kind(),
		)
	}
	return nil
}

func castToError(value reflect.Value) error {
	if value.IsNil() {
		return nil
	}
	return value.Interface().(error)
}

func isError(inType reflect.Type) bool {
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	return inType.Implements(errorInterface)
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
