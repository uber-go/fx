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
	"bytes"
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-validator/validator"
)

const (
	bucketPrimitive = iota
	bucketArray
	bucketObject
	bucketMap
	bucketSlice
)

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

type decoder struct {
	*Value
	m map[interface{}]struct{}
}

func (d *decoder) getGlobalProvider() Provider {
	if d.root == nil {
		return d.provider
	}

	return d.root
}

// Sets value to a primitive type.
func (d *decoder) scalar(childKey string, value reflect.Value, def string) error {
	valueType := value.Type()
	global := d.getGlobalProvider()
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
func (d *decoder) sequence(childKey string, value reflect.Value) error {
	valueType := value.Type()
	global := d.getGlobalProvider()
	destSlice := reflect.MakeSlice(valueType, 0, 4)

	// start looking for child values.
	elementType := derefType(valueType).Elem()
	childKey += _separator

	for ai := 0; ; ai++ {
		arrayKey := childKey + strconv.Itoa(ai)

		// Iterate until we find first missing value.
		if v2 := global.Get(arrayKey); v2.HasValue() {
			val := reflect.New(elementType).Elem()

			// Unmarshal current element.
			if err := d.unmarshal(arrayKey, val, ""); err != nil {
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
		value.Set(destSlice)
	}

	return nil
}

// Set value for the array type
func (d *decoder) array(childKey string, value reflect.Value) error {
	valueType := value.Type()
	global := d.getGlobalProvider()

	// start looking for child values.
	elementType := derefType(valueType).Elem()
	childKey += _separator

	for ai := 0; ai < value.Len(); ai++ {
		arrayKey := childKey + strconv.Itoa(ai)

		// Iterate until we find first missing value.
		if v2 := global.Get(arrayKey); v2.HasValue() {
			val := reflect.New(elementType).Elem()

			// Unmarshal current element.
			if err := d.unmarshal(arrayKey, val, ""); err != nil {
				return err
			}

			value.Index(ai).Set(val)
		} else if value.Index(ai).Kind() == reflect.Struct {
			if err := d.valueStruct(arrayKey, value.Index(ai).Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

// Sets value to a map type.
func (d *decoder) mapping(childKey string, value reflect.Value, def string) error {
	valueType := value.Type()
	global := d.getGlobalProvider()

	v := global.Get(childKey)
	if !v.HasValue() || v.Value() == nil {
		return nil
	}

	val := v.Value()

	// We fallthrough for interface to maps
	if valueType.Kind() == reflect.Interface {
		value.Set(reflect.ValueOf(val))
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
			if err := d.unmarshal(mapKey, itemValue, def); err != nil {
				return err
			}

			destMap.SetMapIndex(reflect.ValueOf(key), itemValue)
		}

		value.Set(destMap)
	}

	return nil
}

// Sets value to an object type.
func (d *decoder) object(childKey string, value reflect.Value) error {
	valueType := value.Type()
	global := d.getGlobalProvider()

	v2 := global.Get(childKey)

	if !v2.HasValue() && valueType.Kind() == reflect.Ptr {
		// in this case we will keep the pointer value as not defined.
		return nil
	}

	if !value.CanSet() {
		return nil
	}

	if valueType.Kind() != reflect.Ptr {
		value = value.Addr()
	}

	if value.IsNil() {
		tmp := reflect.New(value.Type().Elem())
		value.Set(tmp)
	}

	return d.valueStruct(childKey, value.Interface())
}

// Walk through the struct and start asking the providers for values at each key.
//
// - for individual values, we terminate
// - for array values, we start asking for indexes
// - for object values, we recurse.
func (d *decoder) valueStruct(key string, target interface{}) error {
	tarGet := reflect.Indirect(reflect.ValueOf(target))
	targetType := tarGet.Type()
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)

		// Check for the private field
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		fieldName := field.Name
		fieldInfo := getFieldInfo(field)
		if fieldInfo.FieldName != "" {
			fieldName = fieldInfo.FieldName
		}

		if key != "" {
			fieldName = key + _separator + fieldName
		}

		fieldValue := tarGet.Field(i)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type()).Elem())
		}

		if err := d.unmarshal(fieldName, fieldValue, getFieldInfo(field).DefaultValue); err != nil {
			return err
		}
	}

	return validator.Validate(target)
}

// Dispatch un-marshalling functions based on the value type.
func (d *decoder) unmarshal(name string, value reflect.Value, def string) error {
	if value.Type().Comparable() {
		val := value.Interface()
		kind := value.Kind()
		if _, ok := d.m[val]; ok {
			if kind == reflect.Ptr && !value.IsNil() {
				buf := &bytes.Buffer{}
				for k := range d.m {
					fmt.Fprintf(buf, "%+v -> ", k)
				}

				fmt.Fprintf(buf, "%+v", value.Interface())
				return fmt.Errorf("cycles detected: %s", buf.String())
			}
		}

		d.m[val] = struct{}{}
	}

	switch getBucket(value.Type()) {
	case bucketPrimitive:
		return d.scalar(name, value, def)
	case bucketObject:
		return d.object(name, value)
	case bucketArray:
		return d.array(name, value)
	case bucketSlice:
		return d.sequence(name, value)
	case bucketMap:
		return d.mapping(name, value, def)
	}

	return nil
}
