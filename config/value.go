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
	"strings"
	"time"

	"github.com/go-validator/validator"
	"github.com/pkg/errors"
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

func convertValueFromStruct(value interface{}, targetType reflect.Type, fieldType reflect.Type, fieldValue reflect.Value) error {
	// The fieldType is probably a custom type here. We will try and set the fieldValue by
	// the custom type
	// TODO: refactor switch cases into isType functions
	// TODO(alsam) Fix overflows/negatives for unsigned types...
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(int64(value.(int)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		fieldValue.SetUint(uint64(value.(int)))
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(value.(float64))
	case reflect.Bool:
		fieldValue.SetBool(value.(bool))
	case reflect.String:
		fieldValue.SetString(value.(string))
	default:
		return fmt.Errorf("can't convert %v to %v", reflect.TypeOf(value).String(), targetType)
	}
	return nil
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
// TODO(alsam) add check for cycles
// TODO(alsam) now we can populate not only structs. Provide a generic function.
func (cv Value) PopulateStruct(target interface{}) error {
	return cv.valueStruct(cv.key, target)
}

func (cv Value) getGlobalProvider() Provider {
	if cv.root == nil {
		return cv.provider
	}

	return cv.root
}

// Sets value to a primitive type.
func (cv Value) scalar(childKey string, value reflect.Value, def string) error {
	valueType := value.Type()

	global := cv.getGlobalProvider()

	var val interface{}

	if valueType.Kind() == reflect.Ptr {
		if v1 := global.Get(childKey); v1.HasValue() {
			val = v1.Value()
			if val != nil {
				// We cannot assign reflect.ValueOf(Val) to it as is to value.
				// value is a pointer, which currently points to non address.
				// We need to set a new reflect.Value with field type same as the field we are populating
				// before assigning the value (val) parsed from yaml
				if value.IsNil() {
					value.Set(reflect.New(valueType.Elem()))
				}

				// We cannot assign reflect.ValueOf(val) to value as is, when field is a user defined type
				// We need to find the Kind of the custom type and set the value to the specific type
				// that user defined type is defined with.
				kind := valueType.Elem().Kind()
				switch kind {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					value.Elem().SetInt(int64(val.(int)))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					value.Elem().SetUint(uint64(val.(int)))
				case reflect.Float32, reflect.Float64:
					value.Elem().SetFloat(val.(float64))
				case reflect.Bool:
					value.Elem().SetBool(val.(bool))
				case reflect.String:
					value.Elem().SetString(val.(string))
				default:
					value.Elem().Set(reflect.ValueOf(val))
				}
			}
		}

		return nil
	}

	// For primitive values, just get the value and set it into the field
	if v2 := global.Get(childKey); v2.HasValue() {
		val = v2.Value()
	} else if def != "" {
		val = def
	}

	if val != nil {
		// First try to convert primitive type values, if convertValue wasn't able
		// to convert to primitive,try converting the value as a struct value
		if ret, err := convertValue(val, valueType); ret != nil {
			if err != nil {
				return err
			}

			value.Set(reflect.ValueOf(ret))
		} else {
			return convertValueFromStruct(val, value.Type(), valueType, value)
		}
	}

	return nil
}

// Set value for a sequence type
// TODO(alsam) We stop populating sequence on a first nil value. Can we do better?
func (cv Value) sequence(childKey string, fieldValue reflect.Value) error {
	fieldType := fieldValue.Type()
	global := cv.getGlobalProvider()
	destSlice := reflect.MakeSlice(fieldType, 0, 4)

	// start looking for child values.
	elementType := derefType(fieldType).Elem()
	childKey += _separator

	for ai := 0; ; ai++ {
		arrayKey := childKey + strconv.Itoa(ai)

		// Iterate until we find first missing value.
		if v2 := global.Get(arrayKey); v2.HasValue() {
			val := reflect.New(elementType).Elem()

			// Unmarshal current element.
			if err := cv.unmarshal(arrayKey, val, ""); err != nil {
				return err
			}

			// Append element to the slice
			if destSlice.Len() <= ai {
				destSlice = reflect.Append(destSlice, reflect.Zero(elementType))
			}

			destSlice.Index(ai).Set(val)
		} else {
			break
		}
	}

	if destSlice.Len() > 0 {
		fieldValue.Set(destSlice)
	}

	return nil
}

// Sets value to a map type.
func (cv Value) mapping(childKey string, value reflect.Value) error {
	valueType := value.Type()
	global := cv.getGlobalProvider()

	val := global.Get(childKey).Value()

	// We fallthrough for interface to maps
	if valueType.Kind() == reflect.Interface {
		value.Set(reflect.ValueOf(val))
		return nil
	}

	if val == nil {
		return nil
	}

	destMap := reflect.ValueOf(reflect.MakeMap(valueType).Interface())

	// child yamlNode parsed from yaml file is of type map[interface{}]interface{}
	// type casting here makes sure that we are iterating over a parsed map.
	if v, ok := val.(map[interface{}]interface{}); ok {
		childKey += _separator
		for key := range v {
			mapKey := childKey + fmt.Sprintf("%v", key)
			itemValue := reflect.New(valueType.Elem()).Elem()

			// Try to unmarshal value and save it in the map.
			if err := cv.unmarshal(mapKey, itemValue, ""); err != nil {
				return err
			}

			destMap.SetMapIndex(reflect.ValueOf(key), itemValue)
		}

		value.Set(destMap)
	}

	return nil
}

// Sets value to an object type.
func (cv Value) object(childKey string, value reflect.Value) error {
	valueType := value.Type()
	global := cv.getGlobalProvider()

	ntt := derefType(valueType)
	newTarget := reflect.New(ntt)
	v2 := global.Get(childKey)

	if !v2.HasValue() && valueType.Kind() == reflect.Ptr {
		// in this case we will keep the pointer value as not defined.
		return nil
	}

	if err := v2.PopulateStruct(newTarget.Interface()); err != nil {
		return errors.Wrap(err, "unable to populate struct of object target")
	}

	// if the target is not a pointer, deref the value
	// for copy semantics
	if valueType.Kind() != reflect.Ptr {
		newTarget = newTarget.Elem()
	}

	if value.CanSet() {
		value.Set(newTarget)
	}

	return nil
}

// Walk through the struct and start asking the providers for values at each key.
//
// - for individual values, we terminate
// - for array values, we start asking for indexes
// - for object values, we recurse.
func (cv Value) valueStruct(key string, target interface{}) error {
	tarGet := reflect.Indirect(reflect.ValueOf(target))
	targetType := tarGet.Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldName := field.Name
		if fieldName[0] != strings.ToUpper(fieldName)[0] {
			// Skip un-exported fields
			continue
		}

		fieldInfo := getFieldInfo(field)
		if fieldInfo.FieldName != "" {
			fieldName = fieldInfo.FieldName
		}

		if key != "" {
			fieldName = key + _separator + fieldName
		}

		fieldValue := tarGet.Field(i)
		if err := cv.unmarshal(fieldName, fieldValue, getFieldInfo(field).DefaultValue); err != nil {
			return err
		}
	}

	return validator.Validate(target)
}

// Dispatch un-marshalling functions based on value type.
func (cv Value) unmarshal(name string, value reflect.Value, def string) error {
	switch getBucket(value.Type()) {
	case bucketPrimitive:
		return cv.scalar(name, value, def)
	case bucketObject:
		return cv.object(name, value)
	case bucketArray:
	// TODO(alsam) fix array type DRI-12.
	case bucketSlice:
		return cv.sequence(name, value)
	case bucketMap:
		return cv.mapping(name, value)
	}

	return nil
}
