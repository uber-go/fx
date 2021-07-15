// Copyright (c) 2021 Uber Technologies, Inc.
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

package fxevent

import (
	"fmt"
	"io"
	"strings"

	"go.uber.org/fx/internal/fxreflect"
)

// ConsoleLogger is an Fx event logger that attempts to write human-readable
// mesasges to the console.
//
// Use this during development.
type ConsoleLogger struct {
	W io.Writer
}

var _ Logger = (*ConsoleLogger)(nil)

func (l *ConsoleLogger) logf(msg string, args ...interface{}) {
	fmt.Fprintf(l.W, "[Fx] "+msg+"\n", args...)
}

// LogEvent logs the given event to the provided Zap logger.
func (l *ConsoleLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case *LifecycleHookExecuting:
		l.logf("HOOK %s\t\t%s executing (caller: %s)", e.Method, e.FunctionName, e.CallerName)
	case *LifecycleHookExecuted:
		if e.Err != nil {
			l.logf("HOOK %s\t\t%s called by %s failed in %s: %v", e.Method, e.FunctionName, e.CallerName, e.Runtime, e.Err)
		} else {
			l.logf("HOOK %s\t\t%s called by %s ran successfully in %s", e.Method, e.FunctionName, e.CallerName, e.Runtime)
		}
	case *ProvideError:
		l.logf("Error after options were applied: %v", e.Err)
	case *Supply:
		l.logf("SUPPLY\t%v", e.TypeName)
	case *Provide:
		for _, rtype := range e.OutputTypeNames {
			l.logf("PROVIDE\t%v <= %v", rtype, fxreflect.FuncName(e.Constructor))
		}
	case *Invoke:
		l.logf("INVOKE\t\t%s", fxreflect.FuncName(e.Function))
	case *InvokeError:
		l.logf("fx.Invoke(%v) called from:\n%+vFailed: %v",
			fxreflect.FuncName(e.Function), e.Stacktrace, e.Err)
	case *StartError:
		l.logf("ERROR\t\tFailed to start: %v", e.Err)
	case *StopSignal:
		l.logf("%v", strings.ToUpper(e.Signal.String()))
	case *StopError:
		l.logf("ERROR\t\tFailed to stop cleanly: %v", e.Err)
	case *RollbackError:
		l.logf("ERROR\t\tCouldn't roll back cleanly: %v", e.Err)
	case *Rollback:
		l.logf("ERROR\t\tStart failed, rolling back: %v", e.StartErr)
	case *Running:
		l.logf("RUNNING")
	case *CustomLoggerError:
		l.logf("ERROR\t\tFailed to construct custom logger: %v", e.Err)
	case *CustomLogger:
		l.logf("LOGGER\tSetting up custom logger from %v", fxreflect.FuncName(e.Function))
	}
}
