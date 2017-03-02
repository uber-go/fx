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
	raven "github.com/getsentry/raven-go"
	"go.uber.org/zap/zapcore"
)

const (
	_platform          = "go"
	_traceContextLines = 3
	_traceSkipFrames   = 2
)

func ravenSeverity(lvl zapcore.Level) raven.Severity {
	switch lvl {
	case zapcore.DebugLevel:
		return raven.INFO
	case zapcore.InfoLevel:
		return raven.INFO
	case zapcore.WarnLevel:
		return raven.WARNING
	case zapcore.ErrorLevel:
		return raven.ERROR
	case zapcore.DPanicLevel:
		return raven.FATAL
	case zapcore.PanicLevel:
		return raven.FATAL
	case zapcore.FatalLevel:
		return raven.FATAL
	default:
		// Unrecognized levels are fatal.
		return raven.FATAL
	}
}

type client interface {
	Capture(*raven.Packet, map[string]string) (string, chan error)
	Wait()
}

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	DSN   string `yaml:"DSN"`
	Trace *struct {
		Disabled     bool
		SkipFrames   *int `yaml:"skip_frames"`
		ContextLines *int `yaml:"context_lines"`
	}
}

// Build uses the provided configuration to construct a Sentry-backed logging
// core.
func (c Configuration) Build() (zapcore.Core, error) {
	client, err := raven.New(c.DSN)
	if err != nil {
		return zapcore.NewNopCore(), err
	}
	return newCore(c, client, zapcore.ErrorLevel), nil
}

type core struct {
	client
	zapcore.LevelEnabler

	traceEnabled      bool
	traceSkipFrames   int
	traceContextLines int
	fields            map[string]interface{}
}

func newCore(cfg Configuration, c client, enab zapcore.LevelEnabler) *core {
	sentryCore := &core{
		client:            c,
		LevelEnabler:      enab,
		traceEnabled:      true,
		traceSkipFrames:   _traceSkipFrames,
		traceContextLines: _traceContextLines,
		fields:            make(map[string]interface{}),
	}
	t := cfg.Trace
	if t == nil {
		return sentryCore
	}

	if t.Disabled {
		sentryCore.traceEnabled = false
	}
	if t.SkipFrames != nil {
		sentryCore.traceSkipFrames = *t.SkipFrames
	}
	if t.ContextLines != nil {
		sentryCore.traceContextLines = *t.ContextLines
	}
	return sentryCore
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	packet := &raven.Packet{
		Message:   ent.Message,
		Timestamp: raven.Timestamp(ent.Time),
		Level:     ravenSeverity(ent.Level),
		Platform:  _platform,
		Extra:     clone.fields,
	}

	if c.traceEnabled {
		trace := raven.NewStacktrace(c.traceSkipFrames, c.traceContextLines, nil /* app prefixes */)
		if trace != nil {
			packet.Interfaces = append(packet.Interfaces, trace)
		}
	}

	// TODO: Consume the errors channel in the background, incrementing a
	// counter on each failure.
	_, _ = c.Capture(packet, nil)

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		c.Wait()
	}
	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:            c.client,
		LevelEnabler:      c.LevelEnabler,
		traceEnabled:      c.traceEnabled,
		traceSkipFrames:   c.traceSkipFrames,
		traceContextLines: c.traceContextLines,
		fields:            m,
	}
}
