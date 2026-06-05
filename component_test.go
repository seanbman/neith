package fcmp

import (
	"context"
	"testing"
)

func TestFnComponentRenderTargets(t *testing.T) {
	cases := []struct {
		name     string
		fn       func(FnComponent) FnComponent
		tag      string
		targetID string
		mode     renderMode
	}{
		{"append tag", func(f FnComponent) FnComponent { return f.AppendTag("ul") }, "ul", "", renderAppend},
		{"prepend tag", func(f FnComponent) FnComponent { return f.PrependTag("ul") }, "ul", "", renderPrepend},
		{"swap tag inner", func(f FnComponent) FnComponent { return f.SwapTagInner("main") }, "main", "", renderInner},
		{"swap tag outer", func(f FnComponent) FnComponent { return f.SwapTagOuter("main") }, "main", "", renderOuter},
		{"append element", func(f FnComponent) FnComponent { return f.AppendElement("items") }, "", "items", renderAppend},
		{"prepend element", func(f FnComponent) FnComponent { return f.PrependElement("items") }, "", "items", renderPrepend},
		{"swap element inner", func(f FnComponent) FnComponent { return f.SwapElementInner("content") }, "", "content", renderInner},
		{"swap element outer", func(f FnComponent) FnComponent { return f.SwapElementOuter("content") }, "", "content", renderOuter},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := tc.fn(NewFn(context.Background(), nil))

			if fn.dispatch.Function != render {
				t.Fatalf("expected render function, got %q", fn.dispatch.Function)
			}
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

func TestFnComponentRemoveTargets(t *testing.T) {
	cases := []struct {
		name     string
		fn       func(FnComponent) FnComponent
		tag      string
		targetID string
	}{
		{"remove tag", func(f FnComponent) FnComponent { return f.removeTag("dialog") }, "dialog", ""},
		{"remove element", func(f FnComponent) FnComponent { return f.removeElement("modal") }, "", "modal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := tc.fn(NewFn(context.Background(), nil))

			if fn.dispatch.Function != render {
				t.Fatalf("expected render function, got %q", fn.dispatch.Function)
			}
			if fn.dispatch.FnRender.Tag != tc.tag {
				t.Fatalf("expected tag %q, got %q", tc.tag, fn.dispatch.FnRender.Tag)
			}
			if fn.dispatch.FnRender.TargetID != tc.targetID {
				t.Fatalf("expected target id %q, got %q", tc.targetID, fn.dispatch.FnRender.TargetID)
			}
			assertRenderMode(t, fn.dispatch.FnRender, renderRemove)
		})
	}
}

func TestFnComponentClassMutations(t *testing.T) {
	add := NewFn(context.Background(), nil).setClasses("status", false, "active", "visible")
	if add.dispatch.Function != class {
		t.Fatalf("expected class function, got %q", add.dispatch.Function)
	}
	if add.dispatch.FnClass.TargetID != "status" {
		t.Fatalf("expected target id status, got %q", add.dispatch.FnClass.TargetID)
	}
	if add.dispatch.FnClass.Remove {
		t.Fatal("expected add class mutation")
	}
	if got := add.dispatch.FnClass.Names; len(got) != 2 || got[0] != "active" || got[1] != "visible" {
		t.Fatalf("unexpected class names: %v", got)
	}

	remove := NewFn(context.Background(), nil).setClasses("status", true, "active")
	if !remove.dispatch.FnClass.Remove {
		t.Fatal("expected remove class mutation")
	}
}

func TestFnComponentDOMMutations(t *testing.T) {
	fn := NewFn(context.Background(), nil).setDOM("email", "setAttribute", "aria-label", "Email")

	if fn.dispatch.Function != dom {
		t.Fatalf("expected dom function, got %q", fn.dispatch.Function)
	}
	if fn.dispatch.FnDOM.TargetID != "email" {
		t.Fatalf("expected target id email, got %q", fn.dispatch.FnDOM.TargetID)
	}
	if fn.dispatch.FnDOM.Operation != "setAttribute" {
		t.Fatalf("expected setAttribute operation, got %q", fn.dispatch.FnDOM.Operation)
	}
	if fn.dispatch.FnDOM.Name != "aria-label" {
		t.Fatalf("expected aria-label name, got %q", fn.dispatch.FnDOM.Name)
	}
	if fn.dispatch.FnDOM.Value != "Email" {
		t.Fatalf("expected Email value, got %q", fn.dispatch.FnDOM.Value)
	}
}

func assertRenderMode(t *testing.T, render FnRender, mode renderMode) {
	t.Helper()

	if render.Append != (mode == renderAppend) {
		t.Fatalf("append flag mismatch for mode %v", mode)
	}
	if render.Prepend != (mode == renderPrepend) {
		t.Fatalf("prepend flag mismatch for mode %v", mode)
	}
	if render.Inner != (mode == renderInner) {
		t.Fatalf("inner flag mismatch for mode %v", mode)
	}
	if render.Outer != (mode == renderOuter) {
		t.Fatalf("outer flag mismatch for mode %v", mode)
	}
	if render.Remove != (mode == renderRemove) {
		t.Fatalf("remove flag mismatch for mode %v", mode)
	}
}
