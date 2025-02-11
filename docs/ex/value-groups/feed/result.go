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

package feed

import "go.uber.org/fx"

// ResultModule is the module defined in this file.
var ResultModule = fx.Options(
	// --8<-- [start:provide]
	fx.Provide(New),
	// --8<-- [end:provide]
)

// Watcher watches for events.
type Watcher interface{}

type watcher struct{}

// Result is the result of an operation.
// --8<-- [start:result-tagged]
// --8<-- [start:result-init-1]
type Result struct {
	fx.Out

	// ...
	// --8<-- [end:result-init-1]
	Watcher Watcher `group:"watchers"`
	// --8<-- [start:result-init-2]
}

// --8<-- [end:result-init-2]
// --8<-- [end:result-tagged]

// New produces a result object.
// --8<-- [start:new-init-1]
// --8<-- [start:new-watcher]
func New( /* ... */ ) (Result, error) {
	// ...
	// --8<-- [end:new-init-1]
	watcher := &watcher{
		// ...
	}

	// --8<-- [start:new-init-2]
	return Result{
		// ...
		Watcher: watcher,
	}, nil
}

// --8<-- [end:new-watcher]
// --8<-- [end:new-init-2]
