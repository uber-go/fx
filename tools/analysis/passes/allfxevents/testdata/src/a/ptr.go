package a

import (
	"log"

	"go.uber.org/fx/fxevent"
)

type ptrLogger struct{}

func (*ptrLogger) LogEvent(ev fxevent.Event) { // want `\*ptrLogger doesn't handle \[\*Bar \*Foo\]`
	if e, ok := ev.(*fxevent.Baz); ok {
		log.Print(e)
	}

	if e, ok := ev.(*fxevent.Qux); ok {
		log.Print(e)
	}
}
