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

package ulog

import (
	"testing"
	"time"

	"github.com/uber-go/zap"
)

func logfields() []interface{} {
	return []interface{}{
		"int", 123,
		"int64", 123,
		"float", 123.123,
		"string", "four!",
		"bool", true,
		"time", time.Unix(0, 0),
		"duration", time.Second,
		"another string", "done!",
	}
}

func discardedLogger() zap.Logger {
	return zap.New(
		zap.NewJSONEncoder(),
		zap.DiscardOutput,
	)
}

func BenchmarkUlogWithoutFields(b *testing.B) {
	log := Logger()
	log.SetLogger(discardedLogger())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Ulog message")
		}
	})
}

func BenchmarkUlogWithFields(b *testing.B) {
	log := Logger()
	log.SetLogger(discardedLogger())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Ulog message", logfields()...)
		}
	})
}

func BenchmarkUlogLiteWithFields(b *testing.B) {
	log := Logger()
	log.SetLogger(discardedLogger())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Ulog message", "integer", 123, "string", "string")
		}
	})
}

func BenchmarkUlogTextEncoderWithFields(b *testing.B) {
	log := Logger()
	log.SetLogger(zap.New(
		zap.NewTextEncoder(),
		zap.DiscardOutput,
	))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Ulog message", logfields()...)
		}
	})
}

func BenchmarkUlogWithFieldsPreset(b *testing.B) {
	log := Logger(logfields())
	log.SetLogger(discardedLogger())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Ulog message")
		}
	})
}
