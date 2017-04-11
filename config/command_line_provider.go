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

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(val string) error {
	*s = stringSlice(strings.Split(val, ","))
	return nil
}

type commandLineProvider struct {
	p Provider
	NopProvider
}

// NewCommandLineProvider returns a Provider that is using command line parameters as config values.
func NewCommandLineProvider(flags *flag.FlagSet, args []string) Provider {
	if err := flags.Parse(args); err != nil {
		panic(err)
	}

	m := make(map[string]interface{})
	flags.VisitAll(func(f *flag.Flag) {

		// Traverse path elements
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
				curr = map[string]interface{}{}
			}
		}

		// Assign values
		last := path[len(path)-1]
		if ss, ok := f.Value.(*stringSlice); ok {
			slice := []string(*ss)
			tmp := map[string]interface{}{}
			prev[last] = tmp
			for i, str := range slice {
				tmp[fmt.Sprint(i)] = str
			}

			return
		}

		prev[last] = f.Value.String()
	})

	return &commandLineProvider{p: NewStaticProvider(m)}
}

func (c *commandLineProvider) Name() string {
	return "cmd"
}

func (c *commandLineProvider) Get(key string) Value {
	return c.p.Get(key)
}
