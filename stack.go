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

package fx

type stack interface {
	stack() []interface{}
}

func unstack(funcs []interface{}) []interface{} {
	ret := make([]interface{}, 0, len(funcs))
	for _, f := range funcs {
		if stack, ok := f.(stack); ok {
			ret = append(ret, unstack(stack.stack())...)
		} else {
			ret = append(ret, f)
		}
	}
	return ret
}

// Constructors composes multiple constructors into a single Provide-able
// object.
type Constructors []interface{}

func (c Constructors) stack() []interface{} {
	return []interface{}(c)
}

// Starters composes multiple functions into a single Start-able object.
type Starters []interface{}

func (s Starters) stack() []interface{} {
	return []interface{}(s)
}
