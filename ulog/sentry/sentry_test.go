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

package sentry

import (
	"sync"
	"testing"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type spy struct {
	sync.Mutex
	packets []*raven.Packet
	waits   int
}

func (s *spy) Capture(p *raven.Packet, tags map[string]string) (string, chan error) {
	if len(tags) > 0 {
		panic("Sentry integration shouldn't depend on capture-site tags.")
	}
	s.Lock()
	defer s.Unlock()
	s.packets = append(s.packets, p)
	return "", nil
}

func (s *spy) Wait() {
	s.Lock()
	s.waits++
	s.Unlock()
}

func (s *spy) Packets() []*raven.Packet {
	s.Lock()
	defer s.Unlock()
	if len(s.packets) == 0 {
		return nil
	}
	return append([]*raven.Packet{}, s.packets...)
}

func asCore(t testing.TB, iface zapcore.Core) *core {
	c, ok := iface.(*core)
	require.True(t, ok, "Failed to cast Core to sentry *core.")
	return c
}

func TestRavenSeverityMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		z zapcore.Level
		r raven.Severity
	}{
		{zap.DebugLevel, raven.INFO},
		{zap.InfoLevel, raven.INFO},
		{zap.WarnLevel, raven.WARNING},
		{zap.ErrorLevel, raven.ERROR},
		{zap.DPanicLevel, raven.FATAL},
		{zap.PanicLevel, raven.FATAL},
		{zap.FatalLevel, raven.FATAL},
		{zapcore.Level(-42), raven.FATAL},
		{zapcore.Level(100), raven.FATAL},
	}

	for _, tt := range tests {
		assert.Equal(
			t,
			tt.r,
			ravenSeverity(tt.z),
			"Unexpected output converting zap Level %s to raven Severity.", tt.z,
		)
	}
}

func TestCoreWith(t *testing.T) {
	t.Parallel()
	cfg := Configuration{
		DSN: "testdsn",
	}
	// Ensure that we're not sharing map references across generations.
	parent := newCore(cfg, nil, zapcore.ErrorLevel).With([]zapcore.Field{zap.String("parent", "parent")})
	elder := parent.With([]zapcore.Field{zap.String("elder", "elder")})
	younger := parent.With([]zapcore.Field{zap.String("younger", "younger")})

	parentC := asCore(t, parent)
	elderC := asCore(t, elder)
	youngerC := asCore(t, younger)

	assert.Equal(t, map[string]interface{}{
		"parent": "parent",
	}, parentC.fields, "Unexpected fields on parent.")
	assert.Equal(t, map[string]interface{}{
		"parent": "parent",
		"elder":  "elder",
	}, elderC.fields, "Unexpected fields on first child core.")
	assert.Equal(t, map[string]interface{}{
		"parent":  "parent",
		"younger": "younger",
	}, youngerC.fields, "Unexpected fields on second child core.")
}

func TestCoreCheck(t *testing.T) {
	t.Parallel()
	cfg := Configuration{
		DSN: "testdsn",
	}
	core := newCore(cfg, nil, zapcore.ErrorLevel)
	assert.Nil(t, core.Check(zapcore.Entry{}, nil), "Expected nil CheckedEntry for disabled levels.")
	ent := zapcore.Entry{Level: zapcore.ErrorLevel}
	assert.NotNil(t, core.Check(ent, nil), "Expected non-nil CheckedEntry for enabled levels.")
}

func TestConfigWrite(t *testing.T) {
	s := &spy{}
	cfg := Configuration{
		DSN: "testdsn",
	}
	core := newCore(cfg, s, zapcore.ErrorLevel)

	// Write a panic-level message, which should also fire a Sentry event.
	ent := zapcore.Entry{Message: "oh no", Level: zapcore.PanicLevel, Time: time.Now()}
	ce := core.With([]zapcore.Field{zap.String("foo", "bar")}).Check(ent, nil)
	require.NotNil(t, ce, "Expected Check to return non-nil CheckedEntry at enabled levels.")
	ce.Write(zap.String("bar", "baz"))

	// Assert that we wrote and flushed a packet.
	require.Equal(t, 1, len(s.packets), "Expected to write one Sentry packet.")
	assert.Equal(t, 1, s.waits, "Expected to flush buffered events before crashing.")

	// Assert that the captured packet is shaped correctly.
	p := s.packets[0]
	assert.Equal(t, "oh no", p.Message, "Unexpected message in captured packet.")
	assert.Equal(t, raven.FATAL, p.Level, "Unexpected severity in captured packet.")
	require.Equal(t, 1, len(p.Interfaces), "Expected a stacktrace in packet interfaces.")
	trace, ok := p.Interfaces[0].(*raven.Stacktrace)
	require.True(t, ok, "Expected only interface in packet to be a stacktrace.")
	// Trace should contain this test and testing harness main.
	require.Equal(t, 2, len(trace.Frames), "Expected stacktrace to contain at least two frame.")

	frame := trace.Frames[len(trace.Frames)-1]
	assert.Equal(t, "TestConfigWrite", frame.Function, "Expected frame to point to this test function.")
}

func TestConfigBuild(t *testing.T) {
	t.Parallel()
	broken := Configuration{DSN: "invalid"}
	_, err := broken.Build()
	assert.Error(t, err, "Expected invalid DSN to make config building fail.")
}
