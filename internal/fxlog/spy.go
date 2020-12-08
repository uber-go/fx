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

	"go.uber.org/zap"
)

// Spy is an Fx Printer that captures logged statements. It may be used in
// tests of Fx logs.
type Spy struct {
	entries []Entry
}

var _ Logger = &Spy{}

// Log append an Entry.
func (s *Spy) Log(entry Entry) {
	s.entries = append(s.entries, entry)
}

// Messages returns a copy of all captured messages.
func (s *Spy) Messages() []Entry {
	msgs := make([]Entry, len(s.entries))
	copy(msgs, s.entries)
	return msgs
}

// String returns all logged messages as a single newline-delimited string.
func (s *Spy) String() string {
	if len(s.entries) == 0 {
		return ""
	}
	var msg string
	for _, m := range s.entries {
		msg += m.Message
		for _, f := range m.Fields {
			// extra space before f.Key to separate out message.
			msg += fmt.Sprintf("\t%s: %v", f.Key, f.Value)
		}
		if m.Stack != "" {
			msg += fmt.Sprintf(msg, "%q", m.Stack)
		}
		msg += "\n"
	}

	return msg
}

// Fields returns all Fields members of entries.
func (s *Spy) Fields() []zap.Field {
	var fields []zap.Field
	for _, e := range s.entries {
		fields = append(fields, encodeFields(e.Fields, "")...)
	}

	return fields
}

// Reset clears all messages from the Spy.
func (s *Spy) Reset() {
	s.entries = s.entries[:0]
}
