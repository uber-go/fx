// Copyright (c) 2019 Uber Technologies, Inc.
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

// Annotated annotates a constructor provided to Fx with additional options.
// Annotated cannot be used with constructors which produce fx.Out objects.
type Annotated struct {
	// If specified, this will be used as the name of the value. For more
	// information on named values, see the documentation for the fx.Out type.
	//
	// The following,
	//
	//   func NewReadOnlyConnection(...) (*Connection, error)
	//
	//   fx.Provide(fx.Annotated{
	//     Name: "ro",
	//     Target: NewReadOnlyConnection,
	//   })
	//
	// Is equivalent to,
	//
	//   type result struct {
	//     fx.Out
	//
	//     Connection *Connection `name:"ro"`
	//   }
	//
	//   fx.Provide(func(...) (Result, error) {
	//     conn, err := NewReadOnlyConnection(...)
	//     return Result{Connection: conn}, err
	//   })
	Name string

	// Target is the constructor being annotated with fx.Annotated.
	Target interface{}
}
