package fcmp

import (
	"context"
	"testing"
)

func TestEventSubmitter(t *testing.T) {
	ctx := context.WithValue(context.Background(), EventKey, EventListener{
		Submitter: &EventTarget{
			ID:    "save",
			Name:  "intent",
			Value: "save",
		},
	})

	submitter, err := EventSubmitter(ctx)
	if err != nil {
		t.Fatalf("EventSubmitter returned error: %v", err)
	}
	if submitter == nil {
		t.Fatal("EventSubmitter returned nil submitter")
	}
	if submitter.ID != "save" || submitter.Name != "intent" || submitter.Value != "save" {
		t.Fatalf("unexpected submitter: %#v", submitter)
	}

	submitter.ID = "changed"
	second, err := EventSubmitter(ctx)
	if err != nil {
		t.Fatalf("EventSubmitter returned error: %v", err)
	}
	if second.ID != "save" {
		t.Fatalf("EventSubmitter should return a copy, got ID %q", second.ID)
	}
}

func TestEventSubmitterMissingEvent(t *testing.T) {
	_, err := EventSubmitter(context.Background())
	if err != ErrCtxMissingEvent {
		t.Fatalf("expected ErrCtxMissingEvent, got %v", err)
	}
}
