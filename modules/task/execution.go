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
	"context"
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

func (f *fnRegister) setFnNameMap(fnNameMap map[string]interface{}) {
	f.Lock()
	defer f.Unlock()
	f.fnNameMap = fnNameMap
}

func (f *fnRegister) addFn(fnName string, fn interface{}) {
	f.Lock()
	defer f.Unlock()
	f.fnNameMap[fnName] = fn
}

func (f *fnRegister) getFn(fnName string) (interface{}, bool) {
	f.RLock()
	defer f.RUnlock()
	v, ok := f.fnNameMap[fnName]
	return v, ok
}

var fnLookup = fnRegister{fnNameMap: make(map[string]interface{})}

// fnSignature represents a function and its arguments
type fnSignature struct {
	FnName string
	Args   []interface{}
}

// Execute executes the function
func (s *fnSignature) Execute(ctx context.Context) ([]reflect.Value, error) {
	globalBackendStatsClient().TaskExecutionCount().Inc(1)
	targetArgs := make([]reflect.Value, 0, len(s.Args)+1)
	targetArgs = append(targetArgs, reflect.ValueOf(ctx))
	for _, arg := range s.Args {
		targetArgs = append(targetArgs, reflect.ValueOf(arg))
	}
	if fn, ok := fnLookup.getFn(s.FnName); ok {
		fnValue := reflect.ValueOf(fn)
		return fnValue.Call(targetArgs), nil
	}
	return nil, fmt.Errorf("function: %q not found. Did you forget to register?", s.FnName)
}

// Enqueue sends a func before sending to the task queue
func Enqueue(fn interface{}, args ...interface{}) error {
	// Is function registered
	fnName := getFunctionName(fn)
	_, ok := fnLookup.getFn(fnName)
	if !ok {
		globalBackendStatsClient().TaskPublishFail().Inc(1)
		return fmt.Errorf("function: %q not found. Did you forget to register?", fnName)
	}
	fnType := reflect.TypeOf(fn)
	// Validate function against arguments
	if err := validateFnAgainstArgs(fnType, args); err != nil {
		globalBackendStatsClient().TaskPublishFail().Inc(1)
		return err
	}
	// Publish function to the backend
	ctx := args[0].(context.Context)
	s := fnSignature{FnName: fnName, Args: args[1:]}
	sBytes, err := GlobalBackend().Encoder().Marshal(s)
	if err != nil {
		globalBackendStatsClient().TaskPublishFail().Inc(1)
		return errors.Wrap(err, "unable to encode the function or args")
	}
	globalBackendStatsClient().TaskPublishCount().Inc(1)
	return GlobalBackend().Publish(ctx, sBytes)
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
	_, ok := fnLookup.getFn(fnName)
	if ok {
		return nil
	}
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)
		// Interfaces cannot be registered, their implementations should be
		// https://golang.org/pkg/encoding/gob/#Register
		if argType.Kind() != reflect.Interface {
			arg := reflect.Zero(argType).Interface()
			if err := GlobalBackend().Encoder().Register(arg); err != nil {
				return errors.Wrap(err, "unable to register the message for encoding")
			}
		}
	}
	fnLookup.addFn(fnName, fn)
	return nil
}

// Run decodes the message and executes as a task
func Run(ctx context.Context, message []byte) error {
	stopwatch := globalBackendStatsClient().TaskExecutionTime().Start()
	defer stopwatch.Stop()

	var s fnSignature
	if err := GlobalBackend().Encoder().Unmarshal(message, &s); err != nil {
		return errors.Wrap(err, "unable to decode the message")
	}
	// TODO (madhu): Do we need a timeout here?
	retValues, err := s.Execute(ctx)
	if err != nil {
		globalBackendStatsClient().TaskExecuteFail().Inc(1)
		return err
	}
	// Assume only an error will be returned since that is verified before adding to fnRegister
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
	if fnType.NumIn() < 1 {
		return fmt.Errorf(
			"expected at least one argument of type context.Context in function, found %d input arguments",
			fnType.NumIn(),
		)
	}
	if !isContext(fnType.In(0)) {
		return fmt.Errorf("expected first argument to be context.Context but found %s", fnType.In(0))
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
	if v, ok := value.Interface().(error); ok {
		return v
	}
	return fmt.Errorf("expected return value to be error but found: %s", value.Interface())
}

func isContext(inType reflect.Type) bool {
	contextElem := reflect.TypeOf((*context.Context)(nil)).Elem()
	return inType.Implements(contextElem)
}

func isError(inType reflect.Type) bool {
	errorElem := reflect.TypeOf((*error)(nil)).Elem()
	return inType.Implements(errorElem)
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
