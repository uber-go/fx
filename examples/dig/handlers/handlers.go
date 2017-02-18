// Copyright (c) 2017 Uber Technologies, Inc.
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

package handlers

import (
	"io"
	"log"
	"net/http"

	"go.uber.org/fx/dig"
	"go.uber.org/fx/examples/dig/hello"
)

type HelloHandler struct {
	hello.Sayer

	name string
}

func init() {
	err := dig.Register(NewHandler)
	if err != nil {
		log.Fatalf("Failed to inject new handler: %v", err)
	}
}

// DIG injection function
// Parameter of type hello.Sayer signals that we're dependent on it
// and it will be automatically provided by the graph
func NewHandler(s hello.Sayer) *HelloHandler {
	return &HelloHandler{
		Sayer: s,
		name:  "DIG",
	}
}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// voila. Sayer has been auto-populated by DIG and we can be merry
	io.WriteString(w, h.Sayer.SayHello(h.name))
}
