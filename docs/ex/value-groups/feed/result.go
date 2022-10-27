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
	// region provide
	fx.Provide(New),
	// endregion provide
)

// Watcher watches for events.
type Watcher interface{}

type watcher struct{}

// Result is the resul of an operation.
// region result-tagged
// region result-init
type Result struct {
	fx.Out

	// ...
	// endregion result-init
	Watcher Watcher `group:"watchers"`
	// region result-init
}

// endregion result-init
// endregion result-tagged

// New produces a result object.
// region new-init
// region new-watcher
func New( /* ... */ ) (Result, error) {
	// ...
	// endregion new-init
	watcher := &watcher{
		// ...
	}

	// region new-init
	return Result{
		// ...
		Watcher: watcher,
	}, nil
}

// endregion new-watcher
// endregion new-init
