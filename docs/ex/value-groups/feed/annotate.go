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

// AnnotateModule is the module defined in this file.
var AnnotateModule = fx.Options(
	// --8<-- [start:provide-init]
	fx.Provide(
		NewWatcher,
	),
	// --8<-- [end:provide-init]
	// --8<-- [start:provide-wrap-1]
	fx.Provide(
		// --8<-- [start:provide-annotate]
		fx.Annotate(
			NewWatcher,
			// --8<-- [end:provide-wrap-1]
			fx.ResultTags(`group:"watchers"`),
			// --8<-- [start:provide-wrap-2]
		),
		// --8<-- [end:provide-annotate]
	),
	// --8<-- [end:provide-wrap-2]
)

// FileWatcher watches files.
type FileWatcher struct{}

// FileWatcherModule provides a FileWatcher as a Watcher.
var FileWatcherModule = fx.Options(
	fx.Provide(
		// --8<-- [start:annotate-fw]
		fx.Annotate(
			NewFileWatcher,
			fx.As(new(Watcher)),
			fx.ResultTags(`group:"watchers"`),
		),
		// --8<-- [end:annotate-fw]
	),
)

// NewFileWatcher builds a new file watcher.
// --8<-- [start:new-fw-init]
func NewFileWatcher( /* ... */ ) (*FileWatcher, error) {
	// --8<-- [end:new-fw-init]
	return &FileWatcher{
		// ...
	}, nil
}

// NewWatcher builds a watcher.
// --8<-- [start:new-init]
func NewWatcher( /* ... */ ) (Watcher, error) {
	// ...
	// --8<-- [end:new-init]

	return &FileWatcher{
		// ...
	}, nil
}
