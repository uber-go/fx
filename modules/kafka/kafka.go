package kakfa

import (
	"golang.org/x/net/context"
)

type HandlerCreateFunc func(topic string) (Handler, error)

// Handler handles a single transport-level request.
type Handler interface {
	// Handle the given request, writing the response to the given
	// ResponseWriter.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	Handle(
		ctx context.Context,
		message consumer.Message,
	) error
}
