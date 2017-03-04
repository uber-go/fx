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
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// A ValueType is a type-description of a configuration value
type ValueType int

const (
	// Invalid represents an unset or invalid config type
	Invalid ValueType = iota
	// String is, well, you know what it is
	String
	// Integer holds numbers without decimals
	Integer
	// Bool is, well... go check Wikipedia. It's complicated.
	Bool
	// Float is an easy one. They don't sink in pools.
	Float
	// Slice will cut you.
	Slice
	// Dictionary contains words and their definitions
	Dictionary
	// Zero constants
	_float64Zero = float64(0)

	_separator = "."
)

var _typeOfString = reflect.TypeOf("string")

// GetType returns GO type of the provided object
func GetType(value interface{}) ValueType {
	if value == nil {
		return Invalid
	}

	switch value.(type) {
	case string:
		return String
	case int, int32, int64, byte:
		return Integer
	case bool:
		return Bool
	case float64, float32:
		return Float
	default:
		rt := reflect.TypeOf(value)
		switch rt.Kind() {
		case reflect.Slice:
			return Slice
		case reflect.Map:
			return Dictionary
		}
	}

	return Invalid
}

// A Value holds the value of a configuration
type Value struct {
	root         Provider
	provider     Provider
	key          string
	value        interface{}
	found        bool
	defaultValue interface{}
	Timestamp    time.Time
	Type         ValueType
}

// NewValue creates a configuration value from a provider and a set
// of parameters describing the key
func NewValue(
	provider Provider,
	key string,
	value interface{},
	found bool,
	t ValueType,
	timestamp *time.Time,
) Value {
	cv := Value{
		provider:     provider,
		key:          key,
		value:        value,
		defaultValue: nil,
		Type:         t,
		found:        found,
	}

	if timestamp == nil {
		cv.Timestamp = time.Now()
	} else {
		cv.Timestamp = *timestamp
	}
	return cv
}

// Source returns a configuration provider's name
func (cv Value) Source() string {
	if cv.provider == nil {
		return ""
	}
	return cv.provider.Name()
}

// LastUpdated returns when the configuration value was last updated
func (cv Value) LastUpdated() time.Time {
	if !cv.HasValue() {
		return time.Time{} // zero value if never updated?
	}
	return cv.Timestamp
}

// WithDefault creates a shallow copy of the current configuration value and
// sets its default.
func (cv Value) WithDefault(value interface{}) Value {
	// TODO: create a "DefaultProvider" and chain that into the bottom of the current provider:
	//
	// provider = NewProviderGroup(defaultProvider, cv.provider)
	//
	cv2 := cv
	cv2.defaultValue = value
	return cv2
}

// TODO: Support enumerating child keys
// 1. Add a method on Provider to get the child keys for a given prefix
// 2. Implement in the various providers
// 3. Merge the list here
// 4. Return the set of keys.

// ChildKeys returns the child keys
// TODO(ai) what is this and do we need to keep it?
func (cv Value) ChildKeys() []string {
	return nil
}

// String prints out underline value in Value with fmt.Srpintf.
func (cv Value) String() string {
	return fmt.Sprintf("%v", cv.value)
}

// TryAsString attempts to return the configuration value as a string
func (cv Value) TryAsString() (string, bool) {
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf("")); v != nil && err == nil {
		return val.(string), true
	}
	return "", false
}

// TryAsInt attempts to return the configuration value as an int
func (cv Value) TryAsInt() (int, bool) {
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf(0)); v != nil && err == nil {
		return val.(int), true
	}
	switch val := v.(type) {
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float32:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

// TryAsBool attempts to return the configuration value as a bool
func (cv Value) TryAsBool() (bool, bool) {
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf(true)); v != nil && err == nil {
		return val.(bool), true
	}
	return false, false

}

// TryAsFloat attempts to return the configuration value as a float
func (cv Value) TryAsFloat() (float64, bool) {
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf(_float64Zero)); v != nil && err == nil {
		return val.(float64), true
	}
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	default:
		return _float64Zero, false
	}
}

// AsString returns the configuration value as a string, or panics if not
// string-able
func (cv Value) AsString() string {
	s, ok := cv.TryAsString()
	if !ok {
		panic(fmt.Sprintf("Can't convert to string: %v", cv.Value()))
	}
	return s
}

// AsInt returns the configuration value as an int, or panics if not
// int-able
func (cv Value) AsInt() int {
	s, ok := cv.TryAsInt()
	if !ok {
		panic(fmt.Sprintf("Can't convert to int: %T %v", cv.Value(), cv.Value()))
	}
	return s
}

// AsFloat returns the configuration value as an float64, or panics if not
// float64-able
func (cv Value) AsFloat() float64 {
	s, ok := cv.TryAsFloat()
	if !ok {
		panic(fmt.Sprintf("Can't convert to float64: %v", cv.Value()))
	}
	return s
}

// AsBool returns the configuration value as an bool, or panics if not
// bool-able
func (cv Value) AsBool() bool {
	s, ok := cv.TryAsBool()
	if !ok {
		panic(fmt.Sprintf("Can't convert to bool: %v", cv.Value()))
	}
	return s
}

// IsDefault returns whether the return value is the default.
func (cv Value) IsDefault() bool {
	// TODO(ai) what should the semantics be if the provider has a value that's
	// the same as the default value?
	return !cv.found && cv.defaultValue != nil
}

// HasValue returns whether the configuration has a value that can be used
func (cv Value) HasValue() bool {
	return cv.found || cv.IsDefault()
}

// Value returns the underlying configuration's value
func (cv Value) Value() interface{} {
	if cv.found {
		return cv.value
	}
	return cv.defaultValue
}

const (
	bucketPrimitive = iota
	bucketArray
	bucketObject
	bucketMap
	bucketSlice
)

func getBucket(t reflect.Type) int {
	kind := t.Kind()
	if kind == reflect.Ptr {
		kind = t.Elem().Kind()
	}

	switch kind {
	case reflect.Chan:
		fallthrough
	case reflect.Interface:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.Map:
		return bucketMap
	case reflect.Array:
		return bucketArray
	case reflect.Slice:
		return bucketSlice
	case reflect.Struct:
		return bucketObject
	}
	return bucketPrimitive
}

type fieldInfo struct {
	FieldName    string
	DefaultValue string
	Required     bool
}

func getFieldInfo(field reflect.StructField) fieldInfo {
	return fieldInfo{
		FieldName:    field.Tag.Get("yaml"),
		DefaultValue: field.Tag.Get("default"),
	}
}

func derefType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}

// this is a quick-and-dirty conversion method that only handles
// a couple of cases and complains if it finds one it doesn't like.
// needs a bunch more cases.
func convertValue(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	valueType := reflect.TypeOf(value)
	if valueType.AssignableTo(targetType) {
		return value, nil
	} else if targetType == _typeOfString {
		return fmt.Sprintf("%v", value), nil
	}

	switch v := value.(type) {
	case string:
		target := reflect.New(targetType).Interface()
		switch t := target.(type) {
		case *int:
			return strconv.Atoi(v)
		case *bool:
			return strconv.ParseBool(v)
		case *time.Duration:
			return time.ParseDuration(v)
		case encoding.TextUnmarshaler:
			err := t.UnmarshalText([]byte(v))
			// target should have a pointer receiver to be able to change itself based on text
			return reflect.ValueOf(target).Elem().Interface(), err
		}
	}

	return nil, fmt.Errorf("can't convert %v to %v", reflect.TypeOf(value).String(), targetType)
}

// PopulateStruct fills in a struct from configuration
// TODO(alsam) now we can populate not only structs. Provide a generic function.
func (cv Value) PopulateStruct(target interface{}) error {
	d := decoder{Value: &cv, m: make(map[interface{}]struct{})}

	return d.unmarshal(cv.key, reflect.Indirect(reflect.ValueOf(target)), "")
}
