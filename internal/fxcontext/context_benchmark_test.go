package fxcontext

import (
	"context"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
)

func withContext(t *testing.B, f func(fxctx fx.Context)) {
	f(New(context.Background(), service.NullHost()))
}

func BenchmarkContext_StructWrapper(b *testing.B) {
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

func BenchmarkContext_StructWrapperWithValue(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		ctx := context.WithValue(fxctx, _contextLogger, ulog.Logger())

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
				fxCtx := typeUpgrade(fxctx)
				fxCtx.Logger()
			}
		})
	})
}

func BenchmarkContext_WithTypeAssertionWithValue(b *testing.B) {
	withContext(b, func(fxctx fx.Context) {
		ctx := context.WithValue(fxctx, _contextLogger, ulog.Logger())

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				fxCtx := typeUpgrade(ctx)
				fxCtx.Logger()
			}
		})
	})
}

func typeConversion(ctx context.Context) fx.Context {
	return &Context{ctx}
}

func typeUpgrade(ctx context.Context) fx.Context {
	fxctx, ok := ctx.(fx.Context)
	if ok {
		return fxctx
	}
	return &Context{ctx}
}
