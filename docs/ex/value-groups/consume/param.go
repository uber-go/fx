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

// Watcher watches for events.
type Watcher interface{}

// ParamsModule is the module defined in this file.
var ParamsModule = fx.Options(
	// region provide
	fx.Provide(New),
	// endregion provide
)

// Params is a parameter object.
// region param-tagged
// region param-init
type Params struct {
	fx.In

	// ...
	// endregion param-init
	Watchers []Watcher `group:"watchers"`
	// region param-init
}

// endregion param-init
// endregion param-tagged

// Result is a list of watchers.
type Result struct {
	fx.Out

	Emitter *Emitter
}

// New consumes a value group.
// region new-init
// region new-consume
func New(p Params) (Result, error) {
	// ...
	// endregion new-init
	for _, w := range p.Watchers {
		// ...
		// endregion new-consume
		_ = w // unused
	}
	return Result{
		Emitter: &Emitter{ws: p.Watchers},
	}, nil
}
