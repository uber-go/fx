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

package decorator

import (
	"context"
	"fmt"

	"github.com/uber-go/tally"
	"go.uber.org/fx/config"
	"go.uber.org/yarpc/api/transport"
)

// Recovery returns a panic recovery middleware
func Recovery(metrics tally.Scope, cfg config.Provider) Decorator {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) (err error) {
			defer func() {
				err = handlePanic(recover(), err)
			}()
			return next(ctx, req, resw)
		}
	}
}

// RecoveryOneway returns a panic recovery middleware
func RecoveryOneway(metrics tally.Scope, cfg config.Provider) OnewayDecorator {
	return func(next OnewayHandlerFunc) OnewayHandlerFunc {
		return func(ctx context.Context, req *transport.Request) (err error) {
			defer func() {
				err = handlePanic(recover(), err)
			}()
			return next(ctx, req)
		}
	}
}

// handlePanic takes in the result of a recover and returns an error if there
// was a panic
func handlePanic(rec interface{}, existing error) error {
	if rec == nil {
		return existing
	}
	var msg string
	switch rt := rec.(type) {
	case string:
		msg = rt
	case error:
		msg = rt.Error()
	default:
		msg = "unknown reasons for panic"
	}
	return fmt.Errorf("PANIC: %s", msg)
}
