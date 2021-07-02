package fxevent

// This is a partial fxevent package inspired by the real fxevent package,
// but with a fixed list of events we can test against.

type (
	Logger interface{ LogEvent(Event) }
	Event  interface{ event() }
	Foo    struct{}
	Bar    struct{}
	Baz    struct{}
	Qux    struct{}
)

func (*Foo) event() {}
func (*Bar) event() {}
func (*Baz) event() {}
func (*Qux) event() {}
