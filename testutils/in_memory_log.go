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
	"strings"
	"testing"

	"github.com/uber-go/zap"
)

// TestBuffer is a buffer used to test the zap logger
type TestBuffer struct {
	bytes.Buffer
}

// Sync is a nop to conform to zap.WriteSyncer interface
func (b *TestBuffer) Sync() error {
	return nil
}

// Lines returns buffer as array of strings
func (b *TestBuffer) Lines() []string {
	output := strings.Split(b.String(), "\n")
	return output[:len(output)-1]
}

// Stripped returns buffer as a string without the newline
func (b *TestBuffer) Stripped() string {
	return strings.TrimRight(b.String(), "\n")
}

// WithInMemoryLogger creates an in-memory zap logger that can be used in tests
func WithInMemoryLogger(t *testing.T, opts []zap.Option, f func(zap.Logger, *TestBuffer)) {
	sink := &TestBuffer{}
	errSink := &TestBuffer{}

	allOpts := make([]zap.Option, 0, 3+len(opts))
	allOpts = append(allOpts, zap.DebugLevel, zap.Output(sink), zap.ErrorOutput(errSink))
	allOpts = append(allOpts, opts...)
	f(zap.New(zap.NewJSONEncoder(zap.NoTime()), allOpts...), sink)
}
