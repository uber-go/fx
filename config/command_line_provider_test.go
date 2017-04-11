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

package config

import (
	"testing"

	"github.com/ogier/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLineProvider_Roles(t *testing.T) {
	t.Parallel()

	f := pflag.NewFlagSet("", pflag.PanicOnError)
	var s stringSlice
	f.Var(&s, "roles", "")

	c := NewCommandLineProvider(f, []string{`--roles=a,b,c"d"`})
	v := c.Get("roles")
	require.True(t, v.HasValue())
	var roles []string
	require.NoError(t, v.Populate(&roles))
	assert.Equal(t, []string{"a", "b", `c"d"`}, roles)
}

func TestCommandLineProvider_Default(t *testing.T) {
	t.Parallel()

	f := pflag.NewFlagSet("", pflag.PanicOnError)
	f.String("killerFeature", "minesweeper", "Start->Games->Minesweeper")

	c := NewCommandLineProvider(f, nil)
	v := c.Get("killerFeature")
	require.True(t, v.HasValue())
	assert.Equal(t, "minesweeper", v.AsString())
}

func TestCommandLineProvider_Conversion(t *testing.T) {
	t.Parallel()

	f := pflag.NewFlagSet("", pflag.PanicOnError)
	f.String("dozen", "14", " that number of rolls being allowed to the purchaser of a dozen")

	c := NewCommandLineProvider(f, []string{"--dozen=13"})
	v := c.Get("dozen")
	require.True(t, v.HasValue())
	assert.Equal(t, 13, v.AsInt())
}

func TestCommandLineProvider_PanicOnUnknownFlags(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		NewCommandLineProvider(pflag.NewFlagSet("", pflag.ContinueOnError), []string{"--boom"})
	})
}

func TestCommandLineProvider_Name(t *testing.T) {
	t.Parallel()
	p := NewCommandLineProvider(pflag.NewFlagSet("", pflag.PanicOnError), nil)
	assert.Equal(t, "cmd", p.Name())
}

func TestCommandLineProvider_RepeatingArguments(t *testing.T) {
	t.Parallel()

	f := pflag.NewFlagSet("", pflag.PanicOnError)
	f.Int("count", 1, "If I had a million dollars")

	c := NewCommandLineProvider(f, []string{"--count=2", "--count=3"})
	v := c.Get("count")
	require.True(t, v.HasValue())
	assert.Equal(t, "3", v.AsString())
}
