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

package main

import (
	"log"

	"go.uber.org/fx/dig"
	"go.uber.org/fx/examples/dig/handlers"
	"go.uber.org/fx/examples/dig/hello"
	"go.uber.org/fx/modules/uhttp"
	"go.uber.org/fx/service"
)

func main() {
	g := dig.New()
	if err := g.RegisterAll(handlers.NewHandler, hello.NewPoliteSayer); err != nil {
		log.Fatal("Failed to register DIG constructors")
	}

	getHandler := func(service.Host) []uhttp.RouteHandler {
		var h *handlers.HelloHandler
		if err := g.Resolve(&h); err != nil {
			log.Fatal("Failed to resolve the Handler object")
		}

		return []uhttp.RouteHandler{
			uhttp.NewRouteHandler("/", h),
		}
	}

	svc, err := service.WithModule("example", uhttp.New(getHandler)).Build()
	if err != nil {
		log.Fatal("Unable to initialize service", "error", err)
	}

	svc.Start()
}
