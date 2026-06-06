package neith

import (
	"context"
	"encoding/json"
)

// EventData unmarshals the current event's client payload into T.
//
// Handlers normally call EventData from inside a HandleFn. The payload shape
// depends on the browser event: form events usually decode into a map or struct,
// while pointer, keyboard, drag, mouse, and touch events can decode into the
// matching neith event structs.
func EventData[T any](ctx context.Context) (T, error) {
	var t T
	e, err := currentEvent(ctx)
	if err != nil {
		return t, err
	}
	b, err := json.Marshal(e.Data)
	if err != nil {
		return t, err
	}
	err = json.Unmarshal(b, &t)
	return t, err
}

// EventUploads returns file metadata uploaded before the current event dispatch.
//
// File bytes are posted to neith's upload endpoint over HTTP. EventData still
// contains normal form values, and EventUploads exposes the uploaded files.
func EventUploads(ctx context.Context) ([]Upload, error) {
	e, err := currentEvent(ctx)
	if err != nil {
		return nil, err
	}
	return append([]Upload(nil), e.Uploads...), nil
}

// EventSubmitter returns the button or input that submitted a form event.
func EventSubmitter(ctx context.Context) (*EventTarget, error) {
	e, err := currentEvent(ctx)
	if err != nil {
		return nil, err
	}
	if e.Submitter == nil {
		return nil, nil
	}
	submitter := *e.Submitter
	return &submitter, nil
}

func currentEvent(ctx context.Context) (EventListener, error) {
	e, ok := ctx.Value(EventKey).(EventListener)
	if !ok {
		return EventListener{}, ErrCtxMissingEvent
	}
	return e, nil
}
