// Copyright (c) 2020 Uber Technologies, Inc.
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

package fxlog

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxevent"
)

// Spy is an Fx event logger that captures emitted events and/or logged
// statements. It may be used in tests of Fx logs.
type Spy struct {
	testing.TB
	Messages []string
	events   []fxevent.Event
}

// NewSpy is a constructor for Spy logger.
func NewSpy(tb testing.TB) *Spy {
	return &Spy{
		TB: tb,
	}
}

var _ fxevent.Logger = &Spy{}

// LogEvent appends an Event.
func (s *Spy) LogEvent(event fxevent.Event) {
	s.events = append(s.events, event)
}

// Events returns all captured events.
func (s *Spy) Events() []fxevent.Event {
	events := make([]fxevent.Event, len(s.events))
	copy(events, s.events)
	return events
}

// EventTypes returns all captured event types.
func (s *Spy) EventTypes() []string {
	types := make([]string, len(s.events))
	for i, e := range s.events {
		types[i] = reflect.TypeOf(e).Elem().Name()
	}
	return types
}

// Logf logs messages in a specific format to be asserted later.
func (s *Spy) Logf(format string, args ...interface{}) {
	// Log messages are in the format,
	//
	//   2017-10-27T13:03:01.000-0700	DEBUG	your message here	{data here}
	//
	// We strip the first part of these messages because we can't really test
	// for the timestamp from these tests.
	m := fmt.Sprintf(format, args...)
	m = m[strings.IndexByte(m, '\t')+1:]
	s.Messages = append(s.Messages, m)
	s.TB.Log(m)
}

// Reset clears all messages and events from the Spy.
func (s *Spy) Reset() {
	s.events = s.events[:0]
	s.Messages = s.Messages[:0]
}

// AssertMessages asserts that all expected messages were emitted.
func (s *Spy) AssertMessages(msgs ...string) {
	assert.Equal(s.TB, msgs, s.Messages, "logged messages did not match")
}
