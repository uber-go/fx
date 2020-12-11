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

package testutil

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// spy is a TestingT that captures Log requests.
type spy struct{ Logs []string }

func (s *spy) Clear() { s.Logs = nil }

func (s *spy) Logf(msg string, args ...interface{}) {
	s.Logs = append(s.Logs, fmt.Sprintf(msg, args...))
}

func TestWriteSyncer(t *testing.T) {
	var spy spy
	ws := WriteSyncer{T: &spy}

	t.Run("log", func(t *testing.T) {
		spy.Clear()

		io.WriteString(ws, "hello")
		assert.Equal(t, []string{"hello"}, spy.Logs)
	})

	t.Run("sync", func(t *testing.T) {
		assert.NoError(t, ws.Sync())
	})
}
