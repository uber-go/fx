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

import (
	"fmt"
	"reflect"
	"strconv"
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
)

// GetValueType returns GO type of the provided object
func GetValueType(value interface{}) ValueType {
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

// TryAsString attempts to return the configuration value as a string
func (cv Value) TryAsString() (string, bool) {
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf("")); v != nil && err == nil {
		return val.(string), true
	}
	return "", false
}

// TryAsInt attempts to return the confniguration value as an int
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
	f := float64(0)
	v := cv.Value()
	if val, err := convertValue(v, reflect.TypeOf(f)); v != nil && err == nil {
		return val.(float64), true
	}
	return f, false
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
	bucketInvalid   = -1
	bucketPrimative = 0
	bucketArray     = 1
	bucketObject    = 2
	bucketMap       = 3
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
		fallthrough
	case reflect.Slice:
		return bucketArray
	case reflect.Struct:
		return bucketObject
	}
	return bucketPrimative
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

// this is a quick-and-dirty converstion method that only handles
// a couple of cases and complains if it finds one it doen't like.
// needs a bunch more cases.
func convertValue(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	valueType := reflect.TypeOf(value)

	if valueType.AssignableTo(targetType) {
		return value, nil
	} else if targetType.Name() == "string" {
		return fmt.Sprintf("%v", value), nil
	}
	switch v := value.(type) {
	case string:
		switch targetType.Name() {
		case "int":
			return strconv.Atoi(v)
		case "bool":
			return v == "True" || v == "true", nil
		}
	}

	return nil, fmt.Errorf("can't convert %v to %v", valueType, targetType)
}

// PopulateStruct fills in a struct from configuration
func (cv Value) PopulateStruct(target interface{}) error {
	if !cv.HasValue() {
		return nil
	}

	_, err := cv.getValueStruct(cv.key, target)

	return err
}

func (cv Value) getGlobalProvider() Provider {
	if cv.root == nil {
		return cv.provider
	}
	return cv.root
}

func (cv Value) getValueStruct(key string, target interface{}) (interface{}, error) {
	// walk through the struct and start asking the providers for values at each key.
	//
	// - for individual values, we terminate
	// - for array values, we start asking for indexes
	// - for object values, we recurse.
	//

	targetValue := reflect.Indirect(reflect.ValueOf(target))
	targetType := targetValue.Type()
	// if b := getBucket(targetValue); b == bucketInvalid {
	// 	return nil, false, errors.Error("Invalid target object kind")
	// } else if b == bucketPrimative {
	// 	return cc.GetValue(key, def)
	// }

	global := cv.getGlobalProvider()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldType := field.Type
		fieldName := field.Name

		fieldInfo := getFieldInfo(field)
		if fieldInfo.FieldName != "" {
			fieldName = fieldInfo.FieldName
		}

		childKey := fieldName
		if key != "" {
			childKey = key + "." + childKey
		}
		fieldValue := targetValue.Field(i)

		switch getBucket(fieldType) {
		case bucketInvalid:
			continue
		case bucketPrimative:
			var val interface{}

			if fieldType.Kind() == reflect.Ptr {
				if v1 := global.GetValue(childKey); v1.HasValue() {
					val = v1.Value()
					if val != nil {
						// We cannot assign reflect.ValueOf(Val) to it as is to fieldValue.
						// fieldValue is a pointer, which currently points to non address.
						// We need to set a new reflect.Value with field type same as the field we are populating
						// before assigning the value (val) parsed from yaml
						if fieldValue.IsNil() {
							fieldValue.Set(reflect.New(fieldType.Elem()))
						}
						fieldValue.Elem().Set(reflect.ValueOf(val))
					}
				}
				continue
			}

			// For primative values, just get the value and set it into the field
			//
			if v2 := global.GetValue(childKey); v2.HasValue() {
				val = v2.Value()
			} else if fieldInfo.DefaultValue != "" {
				val = fieldInfo.DefaultValue
			}
			if val != nil {
				v3, err := convertValue(val, fieldValue.Type())
				if err != nil {
					return nil, err
				}

				val = v3
				fieldValue.Set(reflect.ValueOf(val))
			}
			continue
		case bucketObject:
			ntt := derefType(fieldType)
			newTarget := reflect.New(ntt)
			if v2 := global.GetValue(childKey); v2.HasValue() {

				if err := v2.PopulateStruct(newTarget.Interface()); err != nil {
					return nil, errors.Wrap(err, "unable to populate struct of object target")
				}
				// if the target is not a pointer, deref the value
				// for copy semantics
				if fieldType.Kind() != reflect.Ptr {
					newTarget = newTarget.Elem()
				}
				fieldValue.Set(newTarget)
			}
		case bucketArray:
			destSlice := reflect.MakeSlice(fieldType, 0, 4)

			// start looking for child values.
			//
			elementType := derefType(fieldType).Elem()
			bucket := getBucket(elementType)

			for ai := 0; ; ai++ {
				arrayKey := fmt.Sprintf("%s.%d", childKey, ai)

				var itemValue interface{}
				switch bucket {
				case bucketPrimative:
					if v2 := global.GetValue(arrayKey); v2.HasValue() {
						itemValue = v2.Value()
					}
				case bucketObject:
					newTarget := reflect.New(elementType)
					if v2 := global.GetValue(arrayKey); v2.HasValue() {
						if err := v2.PopulateStruct(newTarget.Interface()); err != nil {
							return nil, errors.Wrap(err, "unable to populate struct of object")
						}
						itemValue = reflect.Indirect(newTarget).Interface()
					}
				}

				if itemValue != nil {
					// make sure we have the capacity
					if destSlice.Len() <= ai {
						destSlice = reflect.Append(destSlice, reflect.Zero(elementType))
					}

					item := destSlice.Index(ai)
					item.Set(reflect.ValueOf(itemValue))
				} else {
					break
				}
			}
			if destSlice.Len() > 0 {
				fieldValue.Set(destSlice)
			}
		case bucketMap:
			val := global.GetValue(childKey).Value()
			if val != nil {
				destMap := reflect.ValueOf(reflect.MakeMap(fieldType).Interface())

				// child yamlnode parsed from yaml file is of type map[interface{}]interface{}
				// type casting here makes sure that we are iterating over a parsed map.
				v, ok := val.(map[interface{}]interface{})
				if ok {
					for key, value := range v {
						destMap.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
					}
					fieldValue.Set(destMap)
				}
			}
		}
	}
	return target, validator.Validate(target)
}
