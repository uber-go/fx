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

// Client is a client to make requests.
type Client struct{}

// ClientResult holds the result of NewClient.
// --8<-- [start:empty-1]
// --8<-- [start:fxout]
// --8<-- [start:fields]
type ClientResult struct {
	// --8<-- [end:empty-1]
	fx.Out
	// --8<-- [end:fxout]

	Client *Client
	// --8<-- [start:empty-2]
}

// --8<-- [end:empty-2]
// --8<-- [end:fields]

// NewClient builds a new Client.
// --8<-- [start:returnresult]
// --8<-- [start:produce]
func NewClient() (ClientResult, error) {
	// --8<-- [end:returnresult]
	client := &Client{
		// ...
	}
	return ClientResult{Client: client}, nil
}

// --8<-- [end:produce]
