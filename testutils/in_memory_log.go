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

package testutils

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/testutils"
	"go.uber.org/zap/zapcore"
)

// WithInMemoryLogger creates an in-memory zap logger that can be used in tests
func WithInMemoryLogger(t *testing.T, opts []zap.Option, f func(*zap.Logger, *testutils.Buffer)) {
	buffer := &testutils.Buffer{}
	f(
		zap.New(
			zapcore.WriterFacility(
				zapcore.NewJSONEncoder(
					zapcore.EncoderConfig{
						MessageKey:    "msg",
						LevelKey:      "level",
						CallerKey:     "caller",
						StacktraceKey: "stacktrace",
						//EncodeTime:     zapcore.EpochTimeEncoder,
						EncodeDuration: zapcore.SecondsDurationEncoder,
						EncodeLevel:    zapcore.LowercaseLevelEncoder,
					},
				),
				buffer,
				zapcore.DebugLevel,
			),
			opts...,
		),
		buffer,
	)
}
