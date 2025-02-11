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

package resultobject

import "go.uber.org/fx"

// Inspector inspects client state.
type Inspector struct{}

// Result is the result of this module.
// --8<-- [start:full]
// --8<-- [start:start-1]
type Result struct {
	fx.Out

	Client *Client
	// --8<-- [end:start-1]
	Inspector *Inspector
	// --8<-- [start:start-2]
}

// --8<-- [end:start-2]
// --8<-- [end:full]

// New builds a result.
// --8<-- [start:start-3]
func New() (Result, error) {
	client := &Client{
		// ...
	}
	// --8<-- [start:produce]
	return Result{
		Client: client,
		// --8<-- [end:start-3]
		Inspector: &Inspector{
			// ...
		},
		// --8<-- [start:start-4]
	}, nil
	// --8<-- [end:start-4]
	// --8<-- [end:produce]
}
