package fx

import (
	"fmt"
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"
)

func logln(str string) {
	log.Println(prepend(str))
}

func logf(format string, v ...interface{}) {
	log.Printf(prepend(format), v...)
}

func logpanic(err error) {
	log.Panic(prepend(err.Error()))
}

func fatalf(format string, v ...interface{}) {
	log.Fatalf(prepend(format), v...)
}

func prepend(str string) string {
	return fmt.Sprintf("[Fx] %s", str)
}

func fnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func fnLoc(fn interface{}) string {
	mfunc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())

	file, line := mfunc.FileLine(mfunc.Entry())
	file = strings.Replace(file, mainPath(), ".", 1)

	return fmt.Sprintf("%s:%d", file, line)
}

func logProvideType(t interface{}) {
	if reflect.TypeOf(t).Kind() == reflect.Func {
		// LOAD - *p from func main.provide in ./main.go:20
		logf("PROVIDE\t%s()\t\t\t%s", fnName(t), fnLoc(t))
	} else {
		// LOAD - *fx.Lifecycle from func fx.newLifecycle in ./lifecycle.go:25
		logf("PROVIDE\t%s", reflect.TypeOf(t).String())
	}
}

func mainPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}

func caller() string {
	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called Caller()
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "n/a" // proper error her would be better
	}

	// get the info of the actual function that's in the pointer
	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a"
	}

	// return its name
	return fun.Name()
}
