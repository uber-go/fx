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

package client

import (
	"context"
	"testing"

	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/uber-go/tally"

	jconfig "github.com/uber/jaeger-client-go/config"
	"net/http/httptest"
	"go.uber.org/fx/auth"
	"go.uber.org/fx/internal/fxcontext"
	"github.com/opentracing/opentracing-go"
)

// BenchmarkClientFilters/empty-8         	  500000	      3517 ns/op	     256 B/op	       2 allocs/op
// BenchmarkClientFilters/tracing-8       	   20000	     64421 ns/op	    1729 B/op	      29 allocs/op
// BenchmarkClientFilters/auth-8          	   50000	     36574 ns/op	     728 B/op	      16 allocs/op
// BenchmarkClientFilters/default-8       	   10000	    104374 ns/op	    2275 B/op	      43 allocs/op
func BenchmarkClientFilters(b *testing.B) {
	tracer, closer, err := tracing.InitGlobalTracer(&jconfig.Configuration{}, "Test", ulog.NopLogger, tally.NullStatsReporter)
	if err != nil {
		b.Error(err)
	}

	defer closer.Close()
	bm := map[string][]Filter{
		"empty": {},
		"tracing": {tracingFilter()},
		"auth": {authenticationFilter(fakeAuthInfo{_testYaml})},
		"default": {tracingFilter(), authenticationFilter(fakeAuthInfo{_testYaml})},
	}

	for name, filters := range bm {
		chain := newExecutionChain(filters, nopTransport{})
		span := tracer.StartSpan("test_method")
		span.SetBaggageItem(auth.ServiceAuth, "testService")

		ctx := &fxcontext.Context{
			Context: opentracing.ContextWithSpan(context.Background(), span),
		}

		req := httptest.NewRequest("", "http://localhost", nil).WithContext(ctx)

		b.ResetTimer()
		b.Run(name, func (b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := chain.RoundTrip(req); err != nil {
					b.Error(err)
				}
			}
		})
	}
}
