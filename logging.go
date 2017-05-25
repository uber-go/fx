package fx

import (
	"fmt"
	"log"
)

func logln(str string) {
	log.Println(prepend(str))
}

func logf(format string, v ...interface{}) {
	log.Printf(prepend(format), v)
}

func prepend(str string) string {
	return fmt.Sprintf("[Fx] %v", str)
}
