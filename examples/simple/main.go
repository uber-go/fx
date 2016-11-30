// Copyright (c) 2016 Uber Technologies, Inc.
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
	"go.uber.org/fx/modules/rpc"
	"go.uber.org/fx/modules/uhttp"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
	"go.uber.org/yarpc/transport"
)

func registerJSONers(service service.Host) ([]transport.Registrant, error) {
	return nil, nil
}

func main() {
	svc, err := service.WithModules(
		rpc.JSONModule(registerJSONers),
		uhttp.New(registerHTTPers),
	).Build()

	if err != nil {
		ulog.Logger().Fatal("Unable to initialize service: ", err)
	}

	svc.Start(true)
}
