package neith

import "context"

// ContextKey is used to store values in context esp. for event listeners
type ContextKey string

const (
	// EventKey is used to store EventListeners in context
	EventKey ContextKey = "event"
	// RequestKey is used to store http.Request in context
	RequestKey ContextKey = "request"
	// ResponseKey is used to store http.ResponseWriter in context
	ErrorKey ContextKey = "error"
	// dispatchKey is used internally to store dispatchDetails in context
	dispatchKey ContextKey = "__dispatch__"
)

type dispatchDetails struct {
	ConnID    string
	Conn      *conn
	HandlerID string
}

func dispatchFromContext(ctx context.Context) (dispatchDetails, bool) {
	dd, ok := ctx.Value(dispatchKey).(dispatchDetails)
	return dd, ok
}
