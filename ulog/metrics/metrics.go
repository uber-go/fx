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

package metrics

import (
	"errors"

	"github.com/uber-go/tally"
	"github.com/uber-go/zap"
)

var (
	errHookNilEntry = errors.New("can't call a hook on a nil *Entry")
)

// Hook counts the number of logging messages per level
func Hook(s tally.Scope) zap.Hook {
	return zap.Hook(func(e *zap.Entry) error {
		if e == nil {
			return errHookNilEntry
		}

		// TODO: `.Counter()` method is not necessary here and can be optimized
		// There is a finite number of counters here: one for each logging level.
		// If performance per log needs to be optimized this is one of the places
		// to look. Though it won't save much (estimated ~10ns/call)
		s.Counter(e.Level.String()).Inc(1)
		return nil
	})
}
