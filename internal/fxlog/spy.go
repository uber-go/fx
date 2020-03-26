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
	"strings"
)

// Spy is an Fx Printer that captures logged statements. It may be used in
// tests of Fx logs.
type Spy struct {
	msgs []string
}

var _ Printer = &Spy{}

// Printf logs the given message, formatting it in printf-style.
func (s *Spy) Printf(msg string, args ...interface{}) {
	s.msgs = append(s.msgs, fmt.Sprintf(msg, args...))
}

// Messages returns a copy of all captured messages.
func (s *Spy) Messages() []string {
	msgs := make([]string, len(s.msgs))
	copy(msgs, s.msgs)
	return msgs
}

// String returns all logged messages as a single newline-delimited string.
func (s *Spy) String() string {
	if len(s.msgs) == 0 {
		return ""
	}

	// trailing newline because all log entries should have a newline
	// after them.
	return strings.Join(s.msgs, "\n") + "\n"
}

// Reset clears all messages from the Spy.
func (s *Spy) Reset() {
	s.msgs = s.msgs[:0]
}
