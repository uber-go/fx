package util

import (
	"reflect"
)

func FindField(instance interface{}, name *string, rt reflect.Type) (reflect.Value, bool) {
	// walk the fields looking for ones of type Service
	//
	objType := reflect.TypeOf(instance)

	for objType.Kind() == reflect.Ptr || objType.Kind() == reflect.Interface {
		objType = objType.Elem()
	}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.FieldByIndex([]int{i})

		if name != nil && field.Name != *name {
			continue
		}

		if rt != nil && !field.Type.AssignableTo(rt) {
			continue
		}
		// if we got this far, return the value.
		//
		val := reflect.ValueOf(instance)
		return val.FieldByIndex(field.Index), true
	}
	return reflect.Value{}, false
}
