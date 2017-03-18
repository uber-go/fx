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
	"bytes"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// WithInMemoryLogger creates an in-memory *zap.Logger that can be used in
// tests.
func WithInMemoryLogger(t *testing.T, opts []zap.Option, f func(*zap.Logger, *zaptest.Buffer)) {
	sink := &zaptest.Buffer{}
	errSink := zapcore.AddSync(ioutil.Discard)

	allOpts := make([]zap.Option, 0, len(opts)+1)
	allOpts = append(allOpts, zap.ErrorOutput(errSink))
	for _, o := range opts {
		allOpts = append(allOpts, o)
	}
	encoderCfg := zapcore.EncoderConfig{
		LevelKey:    "level",
		MessageKey:  "msg",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}
	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		sink,
		zapcore.DebugLevel,
	)).WithOptions(allOpts...)

	f(log, sink)
}

// LockedBuffer is a thread-safe log buffer for testing
// TODO: This will soon be moved to zap itself, file to be deleted soon
type LockedBuffer struct {
	bytes.Buffer
	zaptest.Syncer
	sync.RWMutex
}

func (b *LockedBuffer) Lines() []string {
	b.RLock()
	defer b.RUnlock()
	output := strings.Split(b.String(), "\n")
	return output[:len(output)-1]
}

func (b *LockedBuffer) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.Write(p)
}

func (b *LockedBuffer) Sync() error {
	b.Lock()
	defer b.Unlock()
	return b.Syncer.Sync()
}

// GetLockedInMemoryLogger creates an in-memory *zap.Logger that can be used in tests.
func GetLockedInMemoryLogger() (*zap.Logger, *LockedBuffer) {
	sink := &LockedBuffer{}

	encoderCfg := zapcore.EncoderConfig{
		LevelKey:    "level",
		MessageKey:  "msg",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}
	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(sink),
		zapcore.DebugLevel,
	))

	return log, sink
}
