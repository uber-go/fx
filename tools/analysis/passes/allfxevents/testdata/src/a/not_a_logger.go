package a

import (
	"fmt"

	"go.uber.org/fx/fxevent"
)

type notALogger struct{}

// Doesn't implement fxevent.Logger because it returns an error. This shouldn't
// cause a diagnostic.

func (*notALogger) LogEvent(ev fxevent.Event) error {
	_, ok := ev.(*fxevent.Foo)
	fmt.Println(ok)
	return nil
}
