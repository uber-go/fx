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

package consume

import "go.uber.org/fx"

// PlainModule is an unannotated NewEmitter.
var PlainModule = fx.Options(
	// region provide-init
	fx.Provide(
		NewEmitter,
	),
	// endregion provide-init
)

// AnnotateModule is the module defined in this file.
var AnnotateModule = fx.Options(
	// region provide-wrap
	fx.Provide(
		// region provide-annotate
		fx.Annotate(
			NewEmitter,
			// endregion provide-wrap
			fx.ParamTags(`group:"watchers"`),
			// region provide-wrap
		),
		// endregion provide-annotate
	),
	// endregion provide-wrap
)

// Emitter emits events
type Emitter struct{ ws []Watcher }

// NewEmitter builds an emitter.
// region new-init
// region new-consume
func NewEmitter(watchers []Watcher) (*Emitter, error) {
	// endregion new-init
	for _, w := range watchers {
		// ...
		// endregion new-consume
		_ = w // unused
	}
	return &Emitter{ws: watchers}, nil
}

// EmitterFromModule is a module that holds EmitterFrom.
var EmitterFromModule = fx.Options(
	fx.Provide(
		// region annotate-variadic
		fx.Annotate(
			EmitterFrom,
			fx.ParamTags(`group:"watchers"`),
		),
		// endregion annotate-variadic
	),
)

// EmitterFrom builds an Emitter from the list of watchers.
// region new-variadic
func EmitterFrom(watchers ...Watcher) (*Emitter, error) {
	// region new-variadic
	return &Emitter{ws: watchers}, nil
}
