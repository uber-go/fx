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
	"fmt"
	"strings"

	flag "github.com/ogier/pflag"
)

// StringSlice is an alias to string slice, that is used to read comma separated flag values.
type StringSlice []string

var _ flag.Value = (*StringSlice)(nil)

// String returns slice elements separated by comma.
func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

// Set splits val using comma as separators.
func (s *StringSlice) Set(val string) error {
	*s = StringSlice(strings.Split(val, ","))
	return nil
}

type commandLineProvider struct {
	Provider
}

// NewCommandLineProvider returns a Provider that is using command line parameters as config values.
// In order to address nested elements one can use dots in flag names which are considered separators.
// One can use StringSlice type to work with a list of comma separated strings.
func NewCommandLineProvider(flags *flag.FlagSet, args []string) Provider {
	if err := flags.Parse(args); err != nil {
		panic(err)
	}

	m := make(map[string]interface{})
	flags.VisitAll(func(f *flag.Flag) {
		prev, last := traversePath(m, f)
		assignValues(prev, last, f.Value)
	})

	return commandLineProvider{Provider: NewStaticProvider(m)}
}

// Assign values to a map element based on value type.
// If value is a StringSlice - create a new map and with keys - indices and values - StringSlice elements.
// Otherwise just assign it's string value.
func assignValues(m map[string]interface{}, key string, value flag.Value) {
	if ss, ok := value.(*StringSlice); ok {
		slice := []string(*ss)
		tmp := make(map[string]interface{}, len(slice))
		m[key] = tmp
		for i, str := range slice {
			tmp[fmt.Sprint(i)] = str
		}

		return
	}

	m[key] = value.String()
}

// Traverse map with the flag name used as path.
func traversePath(m map[string]interface{}, f *flag.Flag) (prev map[string]interface{}, last string) {
	curr, prev := m, m
	path := strings.Split(f.Name, _separator)
	for _, item := range path {
		if _, ok := curr[item]; !ok {
			curr[item] = map[string]interface{}{}
		}

		prev = curr
		if tmp, ok := curr[item].(map[string]interface{}); ok {
			curr = tmp
		} else {
			// This should never happen, because pflag/flag sort flags before calling a visitor,
			// but it is better to be safe then sorry.
			curr = map[string]interface{}{}
		}
	}

	return prev, path[len(path)-1]
}

func (commandLineProvider) Name() string {
	return "cmd"
}
