package fxevent

type (
	Event  struct{}
	Logger interface{ LogEvent(Event) }
)

