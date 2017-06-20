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

package fxtest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/fx"
)

type tb struct{ n int }

func (t *tb) Errorf(string, ...interface{}) {}
func (t *tb) FailNow()                      { t.n++ }

func TestSuccess(t *testing.T) {
	lc := NewLifecycle(t)

	var count int
	lc.Append(fx.Hook{
		OnStart: func() error {
			count++
			return nil
		},
		OnStop: func() error {
			count++
			return nil
		},
	})

	lc.MustStart()
	assert.Equal(t, 1, count, "Expected OnStart hook to run.")
	lc.MustStop()
	assert.Equal(t, 2, count, "Expected OnStop hook to run.")
}

func TestStartFail(t *testing.T) {
	spy := &tb{}
	lc := NewLifecycle(spy)
	lc.Append(fx.Hook{OnStart: func() error { return errors.New("fail") }})

	lc.MustStart()
	assert.Equal(t, 1, spy.n, "Expected lifecycle start to fail.")

	lc.MustStop()
	assert.Equal(t, 1, spy.n, "Expected lifecycle stop to succeed.")
}

func TestStopFail(t *testing.T) {
	spy := &tb{}
	lc := NewLifecycle(spy)
	lc.Append(fx.Hook{OnStop: func() error { return errors.New("fail") }})

	lc.MustStart()
	assert.Equal(t, 0, spy.n, "Expected lifecycle start to succeed.")

	lc.MustStop()
	assert.Equal(t, 1, spy.n, "Expected lifecycle stop to fail.")
}
