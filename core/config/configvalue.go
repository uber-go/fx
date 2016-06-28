package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type ValueType int

const (
	Invalid ValueType = iota
	String
	Integer
	Bool
	Float
	Slice
	Dictionary
)

func getValueType(value interface{}) ValueType {

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

type ConfigurationValue struct {
	root      ConfigurationProvider
	provider  ConfigurationProvider
	key       string
	value     interface{}
	found     bool
	isDefault bool
	Timestamp time.Time
	Type      ValueType
}

func NewConfigurationValue(provider ConfigurationProvider, key string, value interface{}, t ValueType, isDefault bool, timestamp *time.Time) ConfigurationValue {

	cv := ConfigurationValue{
		provider:  provider,
		key:       key,
		value:     value,
		isDefault: isDefault && value != nil,
		Type:      t,
		found:     !isDefault,
	}

	if timestamp == nil {
		cv.Timestamp = time.Now()
	} else {
		cv.Timestamp = *timestamp
	}
	return cv
}

func (cv ConfigurationValue) Provider() string {
	if cv.provider == nil {
		return ""
	}
	return cv.provider.Name()
}

func (cv ConfigurationValue) TryAsString() (string, bool) {
	if val, err := convertValue(cv.value, reflect.TypeOf("")); cv.value != nil && err == nil {
		return val.(string), true
	}
	return "", false
}

func (cv ConfigurationValue) TryAsInt() (int, bool) {
	if val, err := convertValue(cv.value, reflect.TypeOf(0)); cv.value != nil && err == nil {
		return val.(int), true
	}
	return 0, false

}

func (cv ConfigurationValue) TryAsBool() (bool, bool) {
	if val, err := convertValue(cv.value, reflect.TypeOf(true)); cv.value != nil && err == nil {
		return val.(bool), true
	}
	return false, false

}

func (cv ConfigurationValue) TryAsFloat() (float32, bool) {
	f := float32(0)
	if val, err := convertValue(cv.value, reflect.TypeOf(f)); cv.value != nil && err == nil {
		return val.(float32), true
	}
	return f, false
}

func (cv ConfigurationValue) AsString() string {
	s, ok := cv.TryAsString()
	if !ok {
		panic(fmt.Sprintf("Can't convert to string: %v", cv.value))
	}
	return s
}

func (cv ConfigurationValue) AsInt() int {
	s, ok := cv.TryAsInt()
	if !ok {
		panic(fmt.Sprintf("Can't convert to int: %v", cv.value))
	}
	return s
}

func (cv ConfigurationValue) AsFloat() float32 {
	s, ok := cv.TryAsFloat()
	if !ok {
		panic(fmt.Sprintf("Can't convert to float32: %v", cv.value))
	}
	return s
}

func (cv ConfigurationValue) AsBool() bool {
	s, ok := cv.TryAsBool()
	if !ok {
		panic(fmt.Sprintf("Can't convert to bool: %v", cv.value))
	}
	return s
}

func (cv ConfigurationValue) IsDefault() bool {
	return cv.isDefault
}

func (cv ConfigurationValue) HasValue() bool {
	return cv.found
}

func (cv ConfigurationValue) Value() interface{} {
	return cv.value
}

const (
	bucketInvalid   = -1
	bucketPrimative = 0
	bucketArray     = 1
	bucketObject    = 2
)

func getBucket(t reflect.Type) int {
	kind := t.Kind()
	if kind == reflect.Ptr {
		kind = t.Elem().Kind()
	}
	switch kind {
	case reflect.Chan:
	case reflect.Interface:
	case reflect.Func:
	case reflect.Map:
		return bucketInvalid
	case reflect.Array:
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
		Required:     field.Tag.Get("required") == "true",
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

	return nil, errors.New(fmt.Sprintf("Can't convert %v to %v", valueType, targetType))
}

func (cv ConfigurationValue) PopulateStruct(target interface{}) bool {

	if !cv.HasValue() {
		return false
	}

	_, found, _ := cv.GetValueStruct(cv.key, target)

	return found
}

func (cv ConfigurationValue) getGlobalProvider() ConfigurationProvider {
	if cv.root == nil {
		return cv.provider
	}
	return cv.root
}

func (cv ConfigurationValue) GetValueStruct(key string, target interface{}) (interface{}, bool, error) {

	// walk through the struct and start asking the providers for values at each key.
	//
	// - for individual values, we terminate
	// - for array values, we start asking for indexes
	// - for object values, we recurse.
	//

	found := false
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
			// For primative values, just get the value and set it into the field
			//
			if v2 := global.GetValue(childKey, fieldValue.Interface()); v2.HasValue() {
				val = v2.Value()
				found = true
			} else if fieldInfo.Required {
				return nil, false, errors.New(fmt.Sprintf("Field %q must have value for key %q", fieldName, childKey))
			} else if fieldInfo.DefaultValue != "" {
				val = fieldInfo.DefaultValue
			}
			if val != nil {
				if v3, err := convertValue(val, fieldValue.Type()); err != nil {
					return nil, false, err
				} else {
					val = v3
				}
				fieldValue.Set(reflect.ValueOf(val))
			}
			continue
		case bucketObject:
			ntt := derefType(fieldType)
			newTarget := reflect.New(ntt)
			if v2 := global.GetValue(childKey, nil); v2.HasValue() {

				v2.PopulateStruct(newTarget.Interface())

				// if the target is not a pointer, deref the value
				// for copy semantics
				if fieldType.Kind() != reflect.Ptr {
					newTarget = newTarget.Elem()
				}
				fieldValue.Set(newTarget)
				found = true
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
					if v2 := global.GetValue(arrayKey, nil); v2.HasValue() {
						itemValue = v2.Value()
					}
				case bucketObject:
					newTarget := reflect.New(elementType)
					if v2 := global.GetValue(arrayKey, nil); v2.HasValue() {
						v2.PopulateStruct(newTarget.Interface())
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
					found = true
				} else {
					break
				}
			}
			fieldValue.Set(destSlice)
		}

	}

	if found {
		return target, true, nil
	}
	return nil, false, nil
}
