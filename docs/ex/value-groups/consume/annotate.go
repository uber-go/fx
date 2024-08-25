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
	// --8<-- [start:provide-init]
	fx.Provide(
		NewEmitter,
	),
	// --8<-- [end:provide-init]
)

// AnnotateModule is the module defined in this file.
var AnnotateModule = fx.Options(
	// --8<-- [start:provide-wrap-1]
	fx.Provide(
		// --8<-- [start:provide-annotate]
		fx.Annotate(
			NewEmitter,
			// --8<-- [end:provide-wrap-1]
			fx.ParamTags(`group:"watchers"`),
			// --8<-- [start:provide-wrap-2]
		),
		// --8<-- [end:provide-annotate]
	),
	// --8<-- [end:provide-wrap-2]
)

// Emitter emits events
type Emitter struct{ ws []Watcher }

// NewEmitter builds an emitter.
// --8<-- [start:new-init]
// --8<-- [start:new-consume]
func NewEmitter(watchers []Watcher) (*Emitter, error) {
	// --8<-- [end:new-init]
	for _, w := range watchers {
		// ...
		// --8<-- [end:new-consume]
		_ = w // unused
	}
	return &Emitter{ws: watchers}, nil
}

// EmitterFromModule is a module that holds EmitterFrom.
var EmitterFromModule = fx.Options(
	fx.Provide(
		// --8<-- [start:annotate-variadic]
		fx.Annotate(
			EmitterFrom,
			fx.ParamTags(`group:"watchers"`),
		),
		// --8<-- [end:annotate-variadic]
	),
)

// EmitterFrom builds an Emitter from the list of watchers.
// --8<-- [start:new-variadic]
func EmitterFrom(watchers ...Watcher) (*Emitter, error) {
	// --8<-- [start:new-variadic]
	return &Emitter{ws: watchers}, nil
}
