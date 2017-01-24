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

package fxcontext

import (
	"context"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

func withContext(t *testing.B, f func(fxctx fx.Context)) {
	f(New(context.Background(), service.NopHost()))
}

func BenchmarkContext_TypeConversion(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				fxCtx := typeConversion(fxctx)
				fxCtx.Logger()
			}
		})
	})
}

func BenchmarkContext_TypeConversionWithValue(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		ctx := context.WithValue(fxctx, _fxContextStore, store{ulog.Logger()})

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				fxCtx := typeConversion(ctx)
				fxCtx.Logger()
			}
		})
	})
}

func BenchmarkContext_WithTypeAssertion(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				fxCtx := typeAssertion(fxctx)
				fxCtx.Logger()
			}
		})
	})
}

func BenchmarkContext_WithTypeAssertionWithValue(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		ctx := context.WithValue(fxctx, _fxContextStore, store{ulog.Logger()})

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				fxCtx := typeAssertion(ctx)
				fxCtx.Logger()
			}
		})
	})
}

func typeConversion(ctx context.Context) fx.Context {
	return &Context{ctx}
}

func typeAssertion(ctx context.Context) fx.Context {
	fxctx, ok := ctx.(fx.Context)
	if ok {
		return fxctx
	}
	return &Context{ctx}
}
