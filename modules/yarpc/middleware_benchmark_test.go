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

package yarpc

import (
	"context"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/modules/decorator"
	"go.uber.org/fx/service"

	"go.uber.org/yarpc/api/transport"
)

func Benchmark_WithLayeredMiddleware(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		host := service.NopHost()

		m := TransportUnaryMiddleware{
			procedureMap: make(map[string][]decorator.Decorator),
			layerMap:     make(map[string]transport.UnaryHandler),
		}
		decorator := decorator.Recovery(host.Metrics(), config.NewScopedProvider("recovery", host.Config()))
		m.procedureMap["hello"] = append(m.procedureMap["recovery"], decorator)
		for pb.Next() {
			m.Handle(context.Background(), &transport.Request{
				Procedure: "hello",
			}, nil, &fakeHandler{})
			m.Handle(context.Background(), &transport.Request{
				Procedure: "hello",
			}, nil, &fakeHandler{})
		}
	})
}

func Benchmark_SimpleMiddleware(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {

		unary := fakeHandler{}
		for pb.Next() {
			unary.Handle(context.Background(), &transport.Request{
				Procedure: "hello",
			}, nil)
		}
	})
}

type fakeHandler struct {
}

func (f fakeHandler) Handle(
	ctx context.Context,
	_param1 *transport.Request,
	_param2 transport.ResponseWriter,
) error {
	return nil
}
