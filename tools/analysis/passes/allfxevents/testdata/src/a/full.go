package a

import (
	"fmt"

	"go.uber.org/fx/fxevent"
)

type fullLogger struct{}

func (*fullLogger) LogEvent(ev fxevent.Event) {
	switch ev.(type) {
	case *fxevent.Foo, *fxevent.Bar, *fxevent.Baz:
		fmt.Println(ev)
	case *fxevent.Qux:
		fmt.Println(ev)
	}
}
