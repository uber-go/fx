package fxreflect

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"strings"
)

// IsErr returns true when t implements the error interface
func IsErr(t reflect.Type) bool {
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	if t.Implements(errInterface) {
		return true
	}
	return false
}

// ReturnTypes takes a func and returns a slice of string'd types
// TODO instead of duplicating the dig's reflect logic, trying to
// determine which types actually made it into the container, have
// dig return a Result struct which could contain the ProvidedTypes
func ReturnTypes(t interface{}) []string {
	rtypes := []string{}
	fn := reflect.ValueOf(t).Type()

	for i := 0; i < fn.NumOut(); i++ {
		if !IsErr(fn.Out(i)) {
			rtypes = append(rtypes, fn.Out(i).String())
		}
	}

	return rtypes
}

// Caller returns the formatted calling func name
func Caller() string {
	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called Caller()
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "n/a" // TODO return error
	}

	fn := runtime.FuncForPC(fpcs[0] - 1)
	if fn == nil {
		return "n/a" // TODO return error
	}

	return fn.Name()
}

// FuncName returns a funcs formatted name
func FuncName(fn interface{}) string {
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return fmt.Sprintf("%s()", fnName)
}

// FuncLocation returns a funcs formatted relative filepath
func FuncLocation(fn interface{}) string {
	mfunc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())

	file, line := mfunc.FileLine(mfunc.Entry())
	file = strings.Replace(file, mainPath(), ".", 1)

	return fmt.Sprintf("%s:%d", file, line)
}

func mainPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}
