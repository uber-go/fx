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

package service2

import (
	"log"
	"reflect"

	"go.uber.org/dig"
	"go.uber.org/fx/config"
)

// Service foo
type Service struct {
	g  *dig.Graph
	cs []interface{}
}

// Start foo
func (s *Service) Start() {
	// add a bunch of stuff
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, c := range s.cs {
		ctype := reflect.TypeOf(c)
		switch ctype.Kind() {
		case reflect.Func:
			objType := ctype.Out(0)
			s.g.MustResolve(reflect.New(objType).Interface())
		}
	}
}

// Stop foo
func (s *Service) Stop() {
	// close all dig stuff
	log.Println("Stopping...")
}

// New foo
func New(constructors ...interface{}) *Service {
	s := &Service{
		g:  dig.New(),
		cs: constructors,
	}

	s.g.MustRegister(config.DefaultLoader.Load)

	// add a bunch of stuff
	for _, c := range constructors {
		s.g.MustRegister(c)
	}

	return s
}
