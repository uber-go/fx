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
	"reflect"
)

// Spy is an Fx Printer that captures logged statements. It may be used in
// tests of Fx logs.
type Spy struct {
	events []Event
}

var _ Logger = &Spy{}

// LogEvent appends an Event.
func (s *Spy) LogEvent(event Event) {
	s.events = append(s.events, event)
}

// Events returns all captured events.
func (s *Spy) Events() []Event {
	events := make([]Event, len(s.events))
	copy(events, s.events)

	return s.events
}

// EventTypes returns all captured event types.
func (s *Spy) EventTypes() []string {
	types := make([]string, len(s.events))
	for i, e := range s.events {
		types[i] = reflect.TypeOf(e).Elem().Name()
	}

	return types
}

// Reset clears all messages from the Spy.
func (s *Spy) Reset() {
	s.events = s.events[:0]
}
