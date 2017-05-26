package fx

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"go.uber.org/fx/internal/fxreflect"
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

func logProvideType(t interface{}) {
	if reflect.TypeOf(t).Kind() == reflect.Func {
		for _, rtype := range fxreflect.ReturnTypes(t) {
			logf("PROVIDE\t%s <= %s", rtype, fxreflect.FuncName(t))
		}
	} else {
		// LOAD - *fx.Lifecycle from func fx.newLifecycle in ./lifecycle.go:25
		logf("PROVIDE\t%s", reflect.TypeOf(t).String())
	}
}

func logSignal(signal os.Signal) {
	fmt.Println("")
	logln(strings.ToUpper(signal.String()))
}
