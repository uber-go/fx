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
	// region provide-init
	fx.Provide(
		NewWatcher,
	),
	// endregion provide-init
	// region provide-wrap
	fx.Provide(
		// region provide-annotate
		fx.Annotate(
			NewWatcher,
			// endregion provide-wrap
			fx.ResultTags(`group:"watchers"`),
			// region provide-wrap
		),
		// endregion provide-annotate
	),
	// endregion provide-wrap
)

// FileWatcher watches files.
type FileWatcher struct{}

// FileWatcherModule provides a FileWatcher as a Watcher.
var FileWatcherModule = fx.Options(
	fx.Provide(
		// region annotate-fw
		fx.Annotate(
			NewFileWatcher,
			fx.As(new(Watcher)),
			fx.ResultTags(`group:"watchers"`),
		),
		// endregion annotate-fw
	),
)

// NewFileWatcher builds a new file watcher.
// region new-fw-init
func NewFileWatcher( /* ... */ ) (*FileWatcher, error) {
	// endregion new-fw-init
	return &FileWatcher{
		// ...
	}, nil
}

// NewWatcher builds a watcher.
// region new-init
func NewWatcher( /* ... */ ) (Watcher, error) {
	// ...
	// endregion new-init

	return &FileWatcher{
		// ...
	}, nil
}
