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
	"fmt"

	"github.com/uber-go/tally"
	"go.uber.org/zap/zapcore"
)

// Metrics returns a function that counts the number of logs emitted by level.
//
// To register it with a zap.Logger, use zap.Hooks.
func Metrics(s tally.Scope) func(zapcore.Entry) error {
	// Avoid allocating strings and maps in the request path.
	debugC := s.Tagged(map[string]string{"level": "debug"}).Counter("logs")
	infoC := s.Tagged(map[string]string{"level": "info"}).Counter("logs")
	warnC := s.Tagged(map[string]string{"level": "warn"}).Counter("logs")
	errorC := s.Tagged(map[string]string{"level": "error"}).Counter("logs")
	dpanicC := s.Tagged(map[string]string{"level": "dpanic"}).Counter("logs")
	panicC := s.Tagged(map[string]string{"level": "panic"}).Counter("logs")
	fatalC := s.Tagged(map[string]string{"level": "fatal"}).Counter("logs")
	unknownC := s.Tagged(map[string]string{"level": "unknown"}).Counter("logs")

	return func(e zapcore.Entry) error {
		switch e.Level {
		case zapcore.DebugLevel:
			debugC.Inc(1)
		case zapcore.InfoLevel:
			infoC.Inc(1)
		case zapcore.WarnLevel:
			warnC.Inc(1)
		case zapcore.ErrorLevel:
			errorC.Inc(1)
		case zapcore.DPanicLevel:
			dpanicC.Inc(1)
		case zapcore.PanicLevel:
			panicC.Inc(1)
		case zapcore.FatalLevel:
			fatalC.Inc(1)
		default:
			unknownC.Inc(1)
			return fmt.Errorf("unknown log level: %s", e.Level)
		}
		return nil
	}
}
