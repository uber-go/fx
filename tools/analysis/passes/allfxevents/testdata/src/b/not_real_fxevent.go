package b

import (
	"b/fxevent"
	"fmt"
)

type Logger struct{}

var _ fxevent.Logger = Logger{}

func (Logger) LogEvent(ev fxevent.Event) {
	fmt.Println(ev)
}
