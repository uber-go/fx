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
	"errors"
	"time"
)

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

var complexMapYamlV2 = []byte(`
mapStruct:
  oneTrueMap:
    name: poem
    pools:
    - very
    - funny
  nestedStruct:
    additionalData:
`)

type userDefinedTypeInt nestedTypeInt
type nestedTypeInt int64

type userDefinedTypeUInt nestedTypeUInt
type nestedTypeUInt uint32

type userDefinedTypeFloat nestedTypeFloat
type nestedTypeFloat float32

type userDefinedTypeBool nestedTypeBool
type nestedTypeBool bool

type userDefinedTypeString nestedTypeString
type nestedTypeString string

type userDefinedTypeDuration durationStruct

type nestedTypeStructPtr struct {
	TestInt      *userDefinedTypeInt      `yaml:"testInt"`
	TestUInt     *userDefinedTypeUInt     `yaml:"testUInt"`
	TestFloat    *userDefinedTypeFloat    `yaml:"testFloat"`
	TestBool     *userDefinedTypeBool     `yaml:"testBool"`
	TestString   *userDefinedTypeString   `yaml:"testString"`
	TestDuration *userDefinedTypeDuration `yaml:"testDuration"`
}

type typeStructPtr nestedTypeStructPtr

type typeStructStructPtr struct {
	TypeStruct typeStructPtr `yaml:"typeStruct"`
}

type typeStructStructPtrPtr struct {
	TypeStruct *typeStructPtr `yaml:"typeStruct"`
}

type nestedTypeStruct struct {
	TestInt      userDefinedTypeInt      `yaml:"testInt"`
	TestUInt     userDefinedTypeUInt     `yaml:"testUInt"`
	TestFloat    userDefinedTypeFloat    `yaml:"testFloat"`
	TestBool     userDefinedTypeBool     `yaml:"testBool"`
	TestString   userDefinedTypeString   `yaml:"testString"`
	TestDuration userDefinedTypeDuration `yaml:"testDuration"`
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
  testBool: true
  testString: hello
  testDuration:
    seconds: 10s
    minutes: 20m
    hours: 30h
`)

var happyTextUnmarshallerYaml = []byte(`
duckTales:
  protagonist: Scrooge
  pilot: LaunchpadMcQuack
`)

var grumpyTextUnmarshallerYaml = []byte(`
darkwingDuck:
  protagonist: DarkwingDuck
`)

type duckTaleCharacter int

func (d *duckTaleCharacter) UnmarshalText(text []byte) error {
	switch string(text) {
	case "Scrooge":
		*d = scrooge
		return nil
	case "LaunchpadMcQuack":
		*d = launchpadMcQuack
		return nil
	}

	return errors.New("Unknown character: " + string(text))
}

const (
	scrooge duckTaleCharacter = iota
	launchpadMcQuack
)

type duckTales struct {
	Protagonist duckTaleCharacter
	Pilot       duckTaleCharacter
}

type mergeTestData struct {
	description string
	yaml        [][]byte
	expected    map[string]interface{}
}

type firstLevelMerge struct {
	Slice     []string
	Map       map[string]string
	Base      string
	Overwrite string
}

type secondLevelMerge struct {
	Slice     []firstLevelMerge
	Base      firstLevelMerge
	Overwrite firstLevelMerge
}

var baseFirstLevel = firstLevelMerge{
	Map:       map[string]string{"keep": "ok", "override": "updated"},
	Slice:     []string{"Wonder Woman", "Batman"},
	Base:      "MSDOS",
	Overwrite: "Windows 10",
}

var overwriteFirstLevel = firstLevelMerge{
	Map:       map[string]string{"keep": "ok", "override": "updated"},
	Slice:     []string{"Wonder Woman", "Batman"},
	Base:      "UNIX",
	Overwrite: "FreeBSD",
}

var mergeTest = []mergeTestData{

	{
		"First level maps",
		[][]byte{[]byte(`
slice:
- DuckTales
- Darkwingduck
map:
  keep: ok
  override: oldValue
base: UNIX
overwrite: Linux

`),
			[]byte(`
slice:
- Wonder Woman
- Batman
map:
  override: updated
overwrite: FreeBSD
`)},
		map[string]interface{}{
			"": overwriteFirstLevel,
		},
	},
	{
		"Second level structs",
		[][]byte{[]byte(`
slice:
- slice:
  - DuckTales
  - Darkwingduck
  map:
    keep: no
base:
  slice:
  - Wonder Woman
  - Batman
  map:
    keep: ok
    override: updated
  base: MSDOS
  overwrite: Windows 8
overwrite:
  slice:
  - Spider-Man
  - Deadpool
  map:
    keep: ok
    override: oldValue
  overwrite: Linux
`),
			[]byte(`
slice:
- slice:
  - Wonder Woman
  - Batman
  map:
    keep: ok
    override: updated
  base: UNIX
  overwrite: FreeBSD
base:
  map:
    override: updated
  overwrite: Windows 10
overwrite:
  slice:
  - Wonder Woman
  - Batman
  map:
    keep: ok
    override: updated
  base: UNIX
  overwrite: FreeBSD
`)},
		map[string]interface{}{
			"": secondLevelMerge{
				Base:      baseFirstLevel,
				Overwrite: overwriteFirstLevel,
				Slice:     []firstLevelMerge{overwriteFirstLevel}},
			"Base":      baseFirstLevel,
			"Overwrite": overwriteFirstLevel,
		},
	},
	{
		"Empty yamls",
		[][]byte{[]byte(``), []byte(``)},
		map[string]interface{}{
			"": struct{ Field string }{},
		},
	},
	{
		"No overwrite for empty yamls",
		[][]byte{[]byte(`Keep: true`), []byte(``)},
		map[string]interface{}{
			"": struct{ Keep bool }{true},
		},
	},
}
