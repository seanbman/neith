package neith

import (
	"context"
	"testing"
)

func TestEventData(t *testing.T) {
	ctx := context.WithValue(context.Background(), EventKey, EventListener{
		Data: map[string]any{
			"name":   "service call",
			"urgent": true,
		},
	})

	data, err := EventData[struct {
		Name   string `json:"name"`
		Urgent bool   `json:"urgent"`
	}](ctx)
	if err != nil {
		t.Fatalf("EventData returned error: %v", err)
	}
	if data.Name != "service call" || !data.Urgent {
		t.Fatalf("unexpected event data: %#v", data)
	}
}

func TestEventDataMissingEvent(t *testing.T) {
	_, err := EventData[map[string]string](context.Background())
	if err != ErrCtxMissingEvent {
		t.Fatalf("expected ErrCtxMissingEvent, got %v", err)
	}
}

func TestEventUploads(t *testing.T) {
	ctx := context.WithValue(context.Background(), EventKey, EventListener{
		Uploads: []Upload{{
			ID:        "upload-1",
			FieldName: "photo",
			FileName:  "before.jpg",
		}},
	})

	uploads, err := EventUploads(ctx)
	if err != nil {
		t.Fatalf("EventUploads returned error: %v", err)
	}
	if len(uploads) != 1 || uploads[0].FileName != "before.jpg" {
		t.Fatalf("unexpected uploads: %#v", uploads)
	}

	uploads[0].FileName = "after.jpg"
	second, err := EventUploads(ctx)
	if err != nil {
		t.Fatalf("EventUploads returned error: %v", err)
	}
	if second[0].FileName != "before.jpg" {
		t.Fatalf("EventUploads should return a copy, got %q", second[0].FileName)
	}
}

func TestEventUploadsMissingEvent(t *testing.T) {
	_, err := EventUploads(context.Background())
	if err != ErrCtxMissingEvent {
		t.Fatalf("expected ErrCtxMissingEvent, got %v", err)
	}
}

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
