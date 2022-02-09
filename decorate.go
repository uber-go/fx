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
	"strings"

	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxreflect"
)

// Decorate specifies one or more decorator functions to an fx application.
// Decorator functions let users augment objects in the graph. They can take in
// zero or more dependencies that must be provided to the application with fx.Provide,
// and produce one or more values that can be used by other invoked values.
//
// An example decorator is the following function which accepts a value, augments that value,
// and returns the replacement value.
//
//  fx.Decorate(func(log *zap.Logger) *zap.Logger {
//    return log.Named("myapp")
//  })
//
// The following decorator accepts multiple dependencies from the graph, augments and returns
// one of them.
//
//  fx.Decorate(func(log *zap.Logger, cfg *Config) *zap.Logger {
//    return log.Named(cfg.Name)
//  })
//
// All modifications in the object graph due to a decorator are scoped to the fx.Module it was
// specified from.
func Decorate(decorators ...interface{}) Option {
	return decorateOption{
		Targets: decorators,
		Stack:   fxreflect.CallerStack(1, 0),
	}
}

type decorateOption struct {
	Targets []interface{}
	Stack   fxreflect.Stack
}

func (o decorateOption) apply(mod *module) {
	for _, target := range o.Targets {
		mod.decorators = append(mod.decorators, decorator{
			Target: target,
			Stack:  o.Stack,
		})
	}
}

func (o decorateOption) String() string {
	items := make([]string, len(o.Targets))
	for i, f := range o.Targets {
		items[i] = fxreflect.FuncName(f)
	}
	return fmt.Sprintf("fx.Decorate(%s)", strings.Join(items, ", "))
}

// provide is a single decorators used in Fx.
type decorator struct {
	// Constructor provided to Fx. This may be an fx.Annotated.
	Target interface{}

	// Stack trace of where this provide was made.
	Stack fxreflect.Stack
}

func runDecorator(c container, d decorator, opts ...dig.DecorateOption) error {
	decorator := d.Target

	switch decorator := decorator.(type) {
	case annotated:
		dcor, err := decorator.Build()
		if err != nil {
			return fmt.Errorf("fx.Decorate(%v) from:\n%+vFailed: %v", decorator, d.Stack, err)
		}

		if err := c.Decorate(dcor, opts...); err != nil {
			return fmt.Errorf("fx.Decorate(%v) from:\n%+vFailed: %v", decorator, d.Stack, err)
		}
	default:
		if err := c.Decorate(decorator, opts...); err != nil {
			return fmt.Errorf("fx.Decorate(%v) from:\n%+vFailed: %v", decorator, d.Stack, err)
		}
	}
	return nil
}
