package neith

import (
	"context"
	"testing"
)

func TestViewAppliesOptions(t *testing.T) {
	fn := View(context.Background(), HTML("<p>Hello</p>"),
		Label("dashboard"),
		IntoElement("content"),
	)

	if fn.dispatch.Label != "dashboard" {
		t.Fatalf("expected label dashboard, got %q", fn.dispatch.Label)
	}
	if fn.dispatch.Function != render {
		t.Fatalf("expected render function, got %q", fn.dispatch.Function)
	}
	if fn.dispatch.FnRender.TargetID != "content" {
		t.Fatalf("expected target id content, got %q", fn.dispatch.FnRender.TargetID)
	}
	assertRenderMode(t, fn.dispatch.FnRender, renderInner)
}

func TestViewEventHelpers(t *testing.T) {
	handler := func(ctx context.Context) FnComponent {
		return View(ctx, HTML("ok"))
	}

	fn := View(context.Background(), nil, OnSubmit(handler), OnClick(handler))

	if len(fn.dispatch.FnRender.EventListeners) != 0 {
		t.Fatalf("expected no listeners outside dispatch context, got %d", len(fn.dispatch.FnRender.EventListeners))
	}
}

func TestViewRenderTargetOptions(t *testing.T) {
	cases := []struct {
		name     string
		opt      ViewOption
		tag      string
		targetID string
		mode     renderMode
	}{
		{"into tag", IntoTag("main"), "main", "", renderInner},
		{"swap tag outer", SwapTagOuter("main"), "main", "", renderOuter},
		{"append element", AppendToElement("items"), "", "items", renderAppend},
		{"prepend element", PrependToElement("items"), "", "items", renderPrepend},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := View(context.Background(), nil, tc.opt)

			if fn.dispatch.FnRender.Tag != tc.tag {
				t.Fatalf("expected tag %q, got %q", tc.tag, fn.dispatch.FnRender.Tag)
			}
			if fn.dispatch.FnRender.TargetID != tc.targetID {
				t.Fatalf("expected target id %q, got %q", tc.targetID, fn.dispatch.FnRender.TargetID)
			}
			assertRenderMode(t, fn.dispatch.FnRender, tc.mode)
		})
	}
}
