package ulog

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

// Sync is a noop
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
