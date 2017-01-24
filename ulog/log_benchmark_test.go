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

package ulog

import (
	"testing"
	"time"

	"go.uber.org/fx/ulog/sentry"

	"github.com/uber-go/zap"
)

func discardedLogger() zap.Logger {
	return zap.New(
		zap.NewJSONEncoder(),
		zap.DiscardOutput,
	)
}

func withDiscardedLogger(t *testing.B, f func(log Log)) {
	log := Builder().SetLogger(discardedLogger()).Build()
	f(log)
}

func BenchmarkUlogWithoutFields(b *testing.B) {
	withDiscardedLogger(b, func(log Log) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Ulog message")
			}
		})
	})
}

func BenchmarkUlogWithFieldsLogIFace(b *testing.B) {
	withDiscardedLogger(b, func(log Log) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Ulog message",
					"int", 123,
					"int64", 123,
					"float", 123.123,
					"string", "four!",
					"bool", true,
					"time", time.Unix(0, 0),
					"duration", time.Second,
					"another string", "done!")
			}
		})
	})
}

func BenchmarkUlogWithFieldsBaseLoggerStruct(b *testing.B) {
	withDiscardedLogger(b, func(log Log) {
		base := log.(*baseLogger)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				base.Info("Ulog message",
					"int", 123,
					"int64", 123,
					"float", 123.123,
					"string", "four!",
					"bool", true,
					"time", time.Unix(0, 0),
					"duration", time.Second,
					"another string", "done!")
			}
		})
	})
}

func BenchmarkUlogWithFieldsZapLogger(b *testing.B) {
	withDiscardedLogger(b, func(log Log) {
		zapLogger := log.Typed()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				zapLogger.Info("Ulog message",
					zap.Int("int", 123),
					zap.Int64("int64", 123),
					zap.Float64("float", 123.123),
					zap.String("string", "four!"),
					zap.Bool("bool", true),
					zap.Time("time", time.Unix(0, 0)),
					zap.Duration("duration", time.Second),
					zap.String("another string", "done!"))
			}
		})
	})
}

func BenchmarkUlogLiteWithFields(b *testing.B) {
	withDiscardedLogger(b, func(log Log) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Ulog message", "integer", 123, "string", "string")
			}
		})
	})
}

func BenchmarkUlogSentry(b *testing.B) {
	h, _ := sentry.New("")
	l := Builder().SetLogger(discardedLogger()).WithSentryHook(h).Build()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Error("Ulog message", "string", "string")
		}
	})
}

func BenchmarkUlogSentryWith(b *testing.B) {
	h, _ := sentry.New("")
	l := Builder().SetLogger(discardedLogger()).WithSentryHook(h).Build()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.With("foo", "bar").Error("Ulog message", "string", "string")
		}
	})
}
