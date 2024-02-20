// Copyright (c) 2019 Uber Technologies, Inc.
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

import "go.uber.org/dig"

// In can be embedded into a struct to mark it as a parameter struct.
// This allows it to make use of advanced dependency injection features.
// See package documentation for more information.
//
// It's recommended that shared modules use a single parameter struct to
// provide a forward-compatible API:
// adding new optional fields to a struct is backward-compatible,
// so modules can evolve as needs change.
type In = dig.In

// Out is the inverse of In: it marks a struct as a result struct so that
// it can be used with advanced dependency injection features.
// See package documentation for more information.
//
// It's recommended that shared modules use a single result struct to
// provide a forward-compatible API:
// adding new fields to a struct is backward-compatible,
// so modules can produce more outputs as they grow.
type Out = dig.Out
