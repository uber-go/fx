package a

import (
	"fmt"
	"io"

	"go.uber.org/fx/fxevent"
)

type valueLogger struct {
	W io.Writer
}

func (l valueLogger) LogEvent(ev fxevent.Event) { // want `valueLogger doesn't handle \[\*Baz \*Qux\]`
	switch ev.(type) {
	case *fxevent.Foo:
		fmt.Fprintln(l.W, ev)
	case *fxevent.Bar:
		fmt.Fprintln(l.W, ev)
	}
}
