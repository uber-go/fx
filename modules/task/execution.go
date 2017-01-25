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

package task

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"github.com/pkg/errors"
)

// fnRegister allows looking up a function by name
type fnRegister struct {
	fnNameMap map[string]interface{}
	sync.RWMutex
}

var fnLookup = fnRegister{fnNameMap: make(map[string]interface{})}

// fnSignature represents a function and its arguments
type fnSignature struct {
	FnName string
	Args   []interface{}
}

// Execute executes the function
func (s *fnSignature) Execute() ([]reflect.Value, error) {
	targetArgs := make([]reflect.Value, 0, len(s.Args))
	for _, arg := range s.Args {
		targetArgs = append(targetArgs, reflect.ValueOf(arg))
	}
	fnLookup.RLock()
	defer fnLookup.RUnlock()
	if fn, ok := fnLookup.fnNameMap[s.FnName]; ok {
		fnValue := reflect.ValueOf(fn)
		return fnValue.Call(targetArgs), nil
	}
	return nil, fmt.Errorf("function: %q not found. Did you forget to register?", s.FnName)
}

// Enqueue sends a func before sending to the task queue
func Enqueue(fn interface{}, args ...interface{}) error {
	// Is function registered
	fnName := getFunctionName(fn)
	fnLookup.RLock()
	_, ok := fnLookup.fnNameMap[fnName]
	fnLookup.RUnlock()
	if !ok {
		return fmt.Errorf("function: %q not found. Did you forget to register?", fnName)
	}
	fnType := reflect.TypeOf(fn)
	// Validate function against arguments
	if err := validateFnAgainstArgs(fnType, args); err != nil {
		return err
	}
	// Publish function to the backend
	s := fnSignature{FnName: fnName, Args: args}

	sBytes, err := GlobalBackend().Encoder().Marshal(s)
	if err != nil {
		return errors.Wrap(err, "unable to encode the function or args")
	}
	return GlobalBackend().Publish(sBytes, nil)
}

// Register registers a function for async tasks
func Register(fn interface{}) error {
	// Validate that its a function
	fnType := reflect.TypeOf(fn)
	if err := validateFnFormat(fnType); err != nil {
		return err
	}
	// Check if already registered
	fnName := getFunctionName(fn)
	fnLookup.RLock()
	_, ok := fnLookup.fnNameMap[fnName]
	fnLookup.RUnlock()
	if ok {
		return nil
	}
	// Register function types for encoding
	for i := 0; i < fnType.NumIn(); i++ {
		arg := reflect.Zero(fnType.In(i)).Interface()
		if err := GlobalBackend().Encoder().Register(arg); err != nil {
			return errors.Wrap(err, "unable to register the message for encoding")
		}
	}
	fnLookup.Lock()
	defer fnLookup.Unlock()
	fnLookup.fnNameMap[fnName] = fn
	return nil
}

// Run decodes the message and executes as a task
func Run(message []byte) error {
	var s fnSignature
	if err := GlobalBackend().Encoder().Unmarshal(message, &s); err != nil {
		return errors.Wrap(err, "unable to decode the message")
	}
	retValues, err := s.Execute()
	if err != nil {
		return err
	}
	return castToError(retValues[0])
}

func validateFnAgainstArgs(fnType reflect.Type, args []interface{}) error {
	if fnType.NumIn() != len(args) {
		return fmt.Errorf("expected %d function arg(s) but found %d", fnType.NumIn(), len(args))
	}
	for i := 0; i < fnType.NumIn(); i++ {
		argType := reflect.TypeOf(args[i])
		if !argType.AssignableTo(fnType.In(i)) {
			// TODO(madhu): Is it useful to show the arg index or the arg value in the error msg?
			return fmt.Errorf(
				"cannot assign function argument: %d from type: %s to type: %s",
				i+1, argType, fnType.In(i),
			)
		}
	}
	return nil
}

// validateFnFormat verifies that the type is a function type that returns only an error
func validateFnFormat(fnType reflect.Type) error {
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("expected a func as input but was %s", fnType.Kind())
	}
	if fnType.NumOut() != 1 {
		return fmt.Errorf(
			"expected function to return only error but found %d return values", fnType.NumOut(),
		)
	}
	if !isError(fnType.Out(0)) {
		return fmt.Errorf(
			"expected function to return error but found %d", fnType.Out(0).Kind(),
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
