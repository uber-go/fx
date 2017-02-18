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

package dig

// Parent ->
//     Child1 ->
//         Grandchild1
//     Child2 ->
//         Grandchild1
//         Grandchild2
//     Child3 ->
//         GrandchildInt1 (Grandchild1 object)
//         GrandchildInt2 (Grandchild2 object)

type Parent1 struct {
	c1   *Child1
	name string
}

func NewParent1(c1 *Child1) *Parent1 {
	return &Parent1{
		c1:   c1,
		name: "Parent1",
	}
}

type Parent12 struct {
	c1 *Child1
	c2 *Child2

	name string
}

func NewParent12(c1 *Child1, c2 *Child2) *Parent12 {
	return &Parent12{
		c1:   c1,
		c2:   c2,
		name: "Parent12",
	}
}

type Parent123 struct {
	c1 *Child1
	c2 *Child2
	c3 *Child3

	name string
}

func NewParent123(c1 *Child1, c2 *Child2, c3 *Child3) *Parent123 {
	return &Parent123{
		c1:   c1,
		c2:   c2,
		c3:   c3,
		name: "Parent123",
	}
}

type ChildInt1 interface {
	WhatChild1() string
}

type GrandchildInt1 interface {
	WhatGrandchild1() string
}

type GrandchildInt2 interface {
	WhatGrandchild2() string
}

type Child1 struct {
	gc1 *Grandchild1
}

func (c *Child1) WhatChild1() string {
	return "Obj Child1 interface ChildInt1"
}

func NewChild1(gc1 *Grandchild1) *Child1 {
	return &Child1{
		gc1: gc1,
	}
}

type Child2 struct {
	gc1 *Grandchild1
	gc2 *Grandchild2
}

func NewChild2(gc1 *Grandchild1, gc2 *Grandchild2) *Child2 {
	return &Child2{
		gc1: gc1,
		gc2: gc2,
	}
}

type Child3 struct {
	gci1 GrandchildInt1
	gci2 GrandchildInt2
}

func NewChild3(gci1 GrandchildInt1, gci2 GrandchildInt2) *Child3 {
	return &Child3{
		gci1: gci1,
		gci2: gci2,
	}
}

type Grandchild1 struct {
	name string
}

func NewGrandchild1() *Grandchild1 {
	return &Grandchild1{name: "Grandchild1"}
}

func (gc1 *Grandchild1) WhatGrandchild1() string {
	return "Obj Grandchild1, interface WhatGrandchild1"
}

func (gc1 *Grandchild2) WhatGrandchild2() string {
	return "Obj Grandchild1, interface WhatGrandchild2"
}

// Grandchild2 struct does not have a constructor on purpose
// The only way to provide it as a dependency is through object injection
type Grandchild2 struct{}
