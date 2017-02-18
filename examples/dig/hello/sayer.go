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

package hello

import (
	"fmt"
	"log"

	"go.uber.org/fx/dig"
)

func init() {
	err := dig.Register(NewPoliteSayer)
	if err != nil {
		log.Fatalf("Failed to inject a new polite sayer: %v", err)
	}
}

// Sayer returns a string that says hello to the person
type Sayer interface {
	SayHello(person string) string
}

type politeSayer struct{}

func (ps *politeSayer) SayHello(person string) string {
	return fmt.Sprintf("Well hello there %v. How are you?", person)
}

// DIG injection function that provides a polite sayer
// Function takes no parameters, therefore does not register any dependencies
func NewPoliteSayer() Sayer {
	return &politeSayer{}
}
