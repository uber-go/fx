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

package sentry

import (
	"fmt"
	"time"

	"github.com/uber-go/zap"

	raven "github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

const (
	_platform          = "go"
	_traceContextLines = 3
	_traceSkipFrames   = 2
)

var _zapToRavenMap = map[zap.Level]raven.Severity{
	zap.DebugLevel: raven.INFO,
	zap.InfoLevel:  raven.INFO,
	zap.WarnLevel:  raven.WARNING,
	zap.ErrorLevel: raven.ERROR,
	zap.PanicLevel: raven.FATAL,
	zap.FatalLevel: raven.FATAL,
}

// Hook allala
type Hook struct {
	Capturer

	// This is pretty expensive as far as allocations go.
	// No need to copy maps around, especially if they are not going to be used
	// TODO(glib): make this better. We should be able to have an efficient
	// marshaler of this data that won't have us copy maps around
	fields map[string]interface{}

	// Minimum level threshold for sending a Sentry event
	minLevel zap.Level

	// Controls if stack trace should be automatically generated, and how many frames to skip
	traceEnabled      bool
	traceSkipFrames   int
	traceContextLines int
	traceAppPrefixes  []string
}

// Option pattern for Hook creation.
type Option func(l *Hook)

// New Sentry Hook.
func New(dsn string, options ...Option) (*Hook, error) {
	l := &Hook{
		minLevel:          zap.ErrorLevel,
		traceEnabled:      true,
		traceSkipFrames:   _traceSkipFrames,
		traceContextLines: _traceContextLines,
		fields:            make(map[string]interface{}),
	}

	for _, option := range options {
		option(l)
	}

	client, err := raven.New(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create sentry client")
	}

	l.Capturer = &nonBlockingCapturer{Client: client}

	return l, nil
}

// MinLevel provides a minimum level threshold.
// All log messages above the set level are sent to Sentry.
func MinLevel(level zap.Level) Option {
	return func(l *Hook) {
		l.minLevel = level
	}
}

// DisableTraces allows to turn off Stacktrace for sentry packets.
func DisableTraces() Option {
	return func(l *Hook) {
		l.traceEnabled = false
	}
}

// TraceContextLines sets how many lines of code (in on direction) are sent
// with the Sentry packet.
func TraceContextLines(lines int) Option {
	return func(l *Hook) {
		l.traceContextLines = lines
	}
}

// TraceAppPrefixes sets a list of go import prefixes that are considered "in app".
func TraceAppPrefixes(prefixes []string) Option {
	return func(l *Hook) {
		l.traceAppPrefixes = prefixes
	}
}

// TraceSkipFrames sets how many stacktrace frames to skip when sending a
// sentry packet. This is very useful when helper functions are involved.
func TraceSkipFrames(skip int) Option {
	return func(l *Hook) {
		l.traceSkipFrames = skip
	}
}

// Fields stores additional information for each Sentry event.
func Fields(fields map[string]interface{}) Option {
	return func(l *Hook) {
		// TODO(glib): yuck!
		f := make(map[string]interface{})
		for k, v := range fields {
			f[k] = v
		}
		l.fields = f
	}
}

// AppendFields expands the currently stored context of the hook
func (l *Hook) AppendFields(keyvals ...interface{}) {
	for idx := 0; idx < len(keyvals); idx += 2 {
		if key, ok := keyvals[idx].(string); ok {
			val := keyvals[idx+1]
			l.fields[key] = val
		}
	}
}

// CheckAndFire lalala
func (l *Hook) CheckAndFire(lvl zap.Level, msg string, keyvals ...interface{}) {
	fmt.Println(lvl)
	if lvl < l.minLevel {
		return
	}

	// append all the fields from the current log message to the stored ones
	f := make(map[string]interface{}, len(l.fields)+len(keyvals)/2)
	for k, v := range l.fields {
		f[k] = v
	}

	for idx := 0; idx < len(keyvals); idx += 2 {
		if key, ok := keyvals[idx].(string); ok {
			val := keyvals[idx+1]
			f[key] = val
		}
	}

	packet := &raven.Packet{
		Message:   msg,
		Timestamp: raven.Timestamp(time.Now()),
		Level:     _zapToRavenMap[lvl],
		Platform:  _platform,
		Extra:     f,
	}

	if l.traceEnabled {
		trace := raven.NewStacktrace(l.traceSkipFrames, l.traceContextLines, l.traceAppPrefixes)
		if trace != nil {
			packet.Interfaces = append(packet.Interfaces, trace)
		}
	}

	l.Capture(packet)
}
