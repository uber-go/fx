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

package paramobject

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Params defines the parameters of new.
// --8<-- [start:start-1]
// --8<-- [start:full]
type Params struct {
	fx.In

	Config     ClientConfig
	HTTPClient *http.Client
	// --8<-- [end:start-1]
	Logger *zap.Logger `optional:"true"`
	// --8<-- [start:start-2]
}

// --8<-- [end:start-2]
// --8<-- [end:full]

// New builds a new Client.
// --8<-- [start:start-3]
// --8<-- [start:consume]
func New(p Params) (*Client, error) {
	// --8<-- [end:start-3]
	log := p.Logger
	if log == nil {
		log = zap.NewNop()
	}
	// ...
	// --8<-- [end:consume]

	return &Client{log: log}, nil
}
