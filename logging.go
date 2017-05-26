package fx

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	file = strings.Replace(file, gopath(), "$GOPATH", 1)

	return fmt.Sprintf("%s:%d", file, line)
}

func gopath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = defaultGOPATH()
	}
	return gopath
}

func defaultGOPATH() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		def := filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			// Don't set the default GOPATH to GOROOT,
			// as that will trigger warnings from the go tool.
			return ""
		}
		return def
	}
	return ""
}
