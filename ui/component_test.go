package ui

import (
	"context"
	"io"
	"testing"

	"github.com/seanbman/neith"
)

func TestPanelComposesMixedComponents(t *testing.T) {
	got := neith.RenderComponent(Panel(
		Class("settings"),
		Heading("Settings", Level(2)),
		fakeComponent("<p>templ-shaped component</p>"),
		neith.HTML("<hr>"),
		"raw text",
	))

	want := `<section class="n-panel settings"><h2>Settings</h2><p>templ-shaped component</p><hr>raw text</section>`
	if got != want {
		t.Fatalf("unexpected render:\nwant %s\ngot  %s", want, got)
	}
}

func TestFormControlsRenderAttributesAndLabels(t *testing.T) {
	got := neith.RenderComponent(Form(
		TextInput("source",
			Label("Source"),
			ID("source"),
			Value("Billing & invoices"),
			Required(true),
		),
		Select("status",
			Label("Status"),
			Options("ok", "queued"),
			Choices(Choice{Value: "warning", Label: "Needs attention", Selected: true}),
		),
		Button("Save", Type("submit"), Disabled(true)),
	))

	want := `<form><label><span>Source</span><input id="source" name="source" required type="text" value="Billing &amp; invoices"></label><label><span>Status</span><select name="status"><option value="ok">ok</option><option value="queued">queued</option><option value="warning" selected>Needs attention</option></select></label><button disabled type="submit">Save</button></form>`
	if got != want {
		t.Fatalf("unexpected render:\nwant %s\ngot  %s", want, got)
	}
}

func TestTextEscapesContent(t *testing.T) {
	got := neith.RenderComponent(Stack("<script>alert(1)</script>"))
	want := `<div class="n-stack">&lt;script&gt;alert(1)&lt;/script&gt;</div>`
	if got != want {
		t.Fatalf("unexpected render:\nwant %s\ngot  %s", want, got)
	}
}

func TestCustomOptionCanUsePublicConfig(t *testing.T) {
	danger := func() Option {
		return func(c *Config) {
			c.Class("danger")
			c.Attr("data-tone", "danger")
		}
	}

	got := neith.RenderComponent(Button("Delete", danger()))
	want := `<button class="danger" data-tone="danger">Delete</button>`
	if got != want {
		t.Fatalf("unexpected render:\nwant %s\ngot  %s", want, got)
	}
}

type fakeComponent string

func (f fakeComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := io.WriteString(w, string(f))
	return err
}
