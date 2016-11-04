package config

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

type mapStruct struct {
	MyMap map[string]interface{} `yaml:"oneTrueMap"`
}

var simpleMapYaml = []byte(`
mapStruct:
  oneTrueMap:
    one: 1
    two: 2
    three: 3
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
`)
