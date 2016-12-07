// Copyright (c) 2016 Uber Technologies, Inc.
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

import "time"

type emptystruct struct {
	Slice []string
}

var emptyyaml = []byte(`
emptystruct:
  nonexist: true
`)

type pointerStruct struct {
	MyTrueBool  *bool   `yaml:"myTrueBool"`
	MyFalseBool *bool   `yaml:"myFalseBool"`
	MyString    *string `yaml:"myString"`
}

var pointerYaml = []byte(`
pointerStruct:
  myTrueBool: true
  myFalseBool: false
  myString: hello
`)

type nestedStruct struct {
	AdditionalData string `yaml:"additionalData"`
}

type mapStruct struct {
	MyMap        map[string]interface{} `yaml:"oneTrueMap"`
	IntMap       map[string]int         `yaml:"intMap"`
	StringMap    map[string]string      `yaml:"stringMap"`
	NestedStruct nestedStruct           `yaml:"nestedStruct"`
}

var simpleMapYaml = []byte(`
mapStruct:
  oneTrueMap:
    one: 1
    two: 2
    three: 3
  intMap:
    one: 1
    two: 2
  stringMap:
    one: uno
    two: dos
  nestedStruct:
    additionalData: nesteddata
`)

type durationStruct struct {
	Seconds            time.Duration `yaml:"seconds"`
	Minutes            time.Duration `yaml:"minutes"`
	Hours              time.Duration `yaml:"hours"`
	UnparsableDuration time.Duration `yaml:"unparsableDuration"`
}

var durationYaml = []byte(`
durationStruct:
  seconds: 10s
  minutes: 20m
  hours: 30h
`)

var unparsableDurationYaml = []byte(`
durationStruct:
  unparsableDuration: 25thhour
`)

type intKeyMapStruct struct {
	IntKeyMap map[int]string `yaml:"intKeyMap"`
}

var intKeyMapYaml = []byte(`
intKeyMapStruct:
  intKeyMap:
    123: onetwothree
`)

var complexMapYaml = []byte(`
mapStruct:
  oneTrueMap:
    name: heiku
    pools:
    - eee
    - zee
    - pee
    - zeee
    policy:
      makeway: notanoption
  nestedStruct:
    additionalData: nesteddata
`)

type userDefinedTypeInt nestedTypeInt
type nestedTypeInt int64

type userDefinedTypeUInt nestedTypeUInt
type nestedTypeUInt uint32

type userDefinedTypeFloat nestedTypeFloat
type nestedTypeFloat float32

type nestedTypeStruct struct {
	TestInt   *userDefinedTypeInt   `yaml:"testInt"`
	TestUInt  *userDefinedTypeUInt  `yaml:"testUInt"`
	TestFloat *userDefinedTypeFloat `yaml:"testFloat"`
}

type typeStruct nestedTypeStruct

type typeStructStruct struct {
	TypeStruct typeStruct `yaml:"typeStruct"`
}

var typeStructYaml = []byte(`
typeStruct:
  testInt: 123
  testUInt: 456
  testFloat: 123.456
`)
