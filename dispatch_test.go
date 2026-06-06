package neith

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDispatchJSONShape(t *testing.T) {
	dispatch := Dispatch{
		ID:       "dispatch-1",
		Key:      "key-1",
		ConnID:   "conn-1",
		Function: render,
		FnRender: FnRender{
			TargetID: "content",
			Inner:    true,
			HTML:     "<p>Hello</p>",
		},
	}

	b, err := json.Marshal(dispatch)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	body := string(b)
	for _, want := range []string{
		`"function":"render"`,
		`"conn_id":"conn-1"`,
		`"target_id":"content"`,
		`"html":"\u003cp\u003eHello\u003c/p\u003e"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected dispatch JSON to contain %s, got %s", want, body)
		}
	}
	if strings.Contains(body, `"buf"`) || strings.Contains(body, `"conn"`) {
		t.Fatalf("dispatch JSON leaked internal fields: %s", body)
	}
}

func TestFnRenderListenerStrings(t *testing.T) {
	render := FnRender{
		EventListeners: []EventListener{{
			ID:       "listener-1",
			TargetID: "button-1",
			On:       OnClick,
		}},
	}

	listeners := render.listenerStrings()
	for _, want := range []string{
		`"id":"listener-1"`,
		`"target_id":"button-1"`,
		`"on":"click"`,
	} {
		if !strings.Contains(listeners, want) {
			t.Fatalf("expected listener JSON to contain %s, got %s", want, listeners)
		}
	}
}
