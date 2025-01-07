// Copyright (c) 2024 Uber Technologies, Inc.
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

// Evaluate specifies one or more evaluation functions.
// These are functions that accept dependencies from the graph
// and return an fx.Option.
// They may have the following signatures:
//
//	func(...) fx.Option
//	func(...) (fx.Option, error)
//
// These functions are run after provides and decorates.
// The resulting options are applied to the graph,
// and may introduce new provides, invokes, decorates, or evaluates.
//
// The effect of this is that parts of the graph can be dynamically generated
// based on dependency values.
//
// For example, a function with a dependency on a configuration struct
// could conditionally provide different implementations based on the value.
//
//	fx.Evaluate(func(cfg *Config) fx.Option {
//		if cfg.Environment == "production" {
//			return fx.Provide(func(*sql.DB) Repository {
//				return &sqlRepository{db: db}
//			}),
//		} else {
//			return fx.Provide(func() Repository {
//				return &memoryRepository{}
//			})
//		}
//	})
//
// This is different from a normal provide that inspects the configuration
// because the dependency on '*sql.DB' is completely absent in the graph
// if the configuration is not "production".
func Evaluate(fns ...any) Option {
	return evaluateOption{
		Targets: fns,
		Stack:   fxreflect.CallerStack(1, 0),
	}
}

type evaluateOption struct {
	Targets []any
	Stack   fxreflect.Stack
}

func (o evaluateOption) apply(mod *module) {
	for _, target := range o.Targets {
		mod.evaluates = append(mod.evaluates, evaluate{
			Target: target,
			Stack:  o.Stack,
		})
	}
}

func (o evaluateOption) String() string {
	items := make([]string, len(o.Targets))
	for i, target := range o.Targets {
		items[i] = fxreflect.FuncName(target)
	}
	return fmt.Sprintf("fx.Evaluate(%s)", strings.Join(items, ", "))
}

type evaluate struct {
	Target any
	Stack  fxreflect.Stack
}

func runEvaluate(m *module, e evaluate) (err error) {
	target := e.Target
	defer func() {
		if err != nil {
			err = fmt.Errorf("fx.Evaluate(%v) from:\n%+vFailed: %w", target, e.Stack, err)
		}
	}()

	// target is a function returning (Option, error).
	// Use reflection to build a function with the same parameters,
	// and invoke that in the container.
	targetV := reflect.ValueOf(target)
	targetT := targetV.Type()
	inTypes := make([]reflect.Type, targetT.NumIn())
	for i := range targetT.NumIn() {
		inTypes[i] = targetT.In(i)
	}
	outTypes := []reflect.Type{reflect.TypeOf((*error)(nil)).Elem()}

	// TODO: better way to extract information from the container
	var opt Option
	invokeFn := reflect.MakeFunc(
		reflect.FuncOf(inTypes, outTypes, false),
		func(args []reflect.Value) []reflect.Value {
			out := targetV.Call(args)
			switch len(out) {
			case 2:
				if err, _ := out[1].Interface().(error); err != nil {
					return []reflect.Value{reflect.ValueOf(err)}
				}

				fallthrough
			case 1:
				opt, _ = out[0].Interface().(Option)

			default:
				panic("TODO: validation")
			}

			return []reflect.Value{
				reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()),
			}
		},
	).Interface()
	if err := m.scope.Invoke(invokeFn); err != nil {
		return err
	}

	if opt == nil {
		// Assume no-op.
		return nil
	}

	opt.apply(m)
	return nil
}
