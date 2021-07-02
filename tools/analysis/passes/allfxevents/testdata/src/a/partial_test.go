package a

import (
	"fmt"

	"go.uber.org/fx/fxevent"
)

// This logger intentionally doesn't handle everything. We don't expect any
// diagnostics reported for it because it's in a test file.
type partialLogger struct{}

func (partialLogger) LogEvent(ev fxevent.Event) {
	_, ok := ev.(*fxevent.Foo)
	fmt.Println(ok)
}
