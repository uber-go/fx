// Copyright (c) 2023 Uber Technologies, Inc.
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

type privateOption struct{}

// Private is an [Option] that will mark the scope of a module as private.
// When a module is private, all of its provided and supplied contents
// (including those of inner modules that the private modules contains)
// cannot be accessed by outer modules that contain the private module.
//
//	fx.New(
//		fx.Module("SubModule",
//			fx.Private,
//			fx.Supply(5),
//			fx.Provide(func() string { return "b"}),
//			fx.Invoke(a int, b str) {}, // Runs w/ a = 5, b = "b"
//		),
//		fx.Invoke(func(a int) {}), // This will fail
//		fx.Invoke(func(b str) {}), // This will also fail
//	)
//
// Private can also be used directly as an argument in a call to [Provide] to
// mark only those provided functions as private, rather than a whole module.
//
//	fx.New(
//		fx.Module("SubModule",
//		fx.Provide(func() int { return 0 }, fx.Private),
//			fx.Provide(func() string { return "b" }),
//		),
//		fx.Invoke(func(a int) {}), // This will fail
//		fx.Invoke(func(b str) {}), // This will not
//	)
var Private = privateOption{}

func (o privateOption) String() string {
	return "fx.Private"
}

func (o privateOption) apply(mod *module) {}
