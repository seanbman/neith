package neith

import (
	"context"
	"testing"
)

func TestFnComponentRenderTargets(t *testing.T) {
	cases := []struct { name string; fn func(FnComponent) FnComponent; tag, targetID string; mode renderMode }{
		{"append tag", func(f FnComponent) FnComponent { return f.AppendTag("ul") }, "ul", "", renderAppend},
		{"prepend tag", func(f FnComponent) FnComponent { return f.PrependTag("ul") }, "ul", "", renderPrepend},
		{"swap tag inner", func(f FnComponent) FnComponent { return f.SwapTagInner("main") }, "main", "", renderInner},
		{"swap tag outer", func(f FnComponent) FnComponent { return f.SwapTagOuter("main") }, "main", "", renderOuter},
		{"append element", func(f FnComponent) FnComponent { return f.AppendElement("items") }, "", "items", renderAppend},
		{"prepend element", func(f FnComponent) FnComponent { return f.PrependElement("items") }, "", "items", renderPrepend},
		{"swap element inner", func(f FnComponent) FnComponent { return f.SwapElementInner("content") }, "", "content", renderInner},
		{"swap element outer", func(f FnComponent) FnComponent { return f.SwapElementOuter("content") }, "", "content", renderOuter},
	}
	for _, tc := range cases { t.Run(tc.name, func(t *testing.T) { fn:=tc.fn(NewFn(context.Background(),nil)); if fn.dispatch.Function!=render { t.Fatalf("expected render function, got %q",fn.dispatch.Function) }; if fn.dispatch.FnRender.Tag!=tc.tag { t.Fatalf("expected tag %q, got %q",tc.tag,fn.dispatch.FnRender.Tag) }; if fn.dispatch.FnRender.TargetID!=tc.targetID { t.Fatalf("expected target id %q, got %q",tc.targetID,fn.dispatch.FnRender.TargetID) }; assertRenderMode(t,fn.dispatch.FnRender,tc.mode) }) }
}

func TestFnComponentRemoveTargets(t *testing.T) {
	cases:=[]struct{name string; fn func(FnComponent)FnComponent; tag,targetID string}{
		{"remove tag",func(f FnComponent)FnComponent{return f.removeTag("dialog")},"dialog",""},
		{"remove element",func(f FnComponent)FnComponent{return f.removeElement("modal")},"","modal"},
	}
	for _,tc:=range cases { t.Run(tc.name,func(t *testing.T){ fn:=tc.fn(NewFn(context.Background(),nil)); if fn.dispatch.Function!=render {t.Fatalf("expected render function, got %q",fn.dispatch.Function)}; if fn.dispatch.FnRender.Tag!=tc.tag {t.Fatalf("expected tag %q, got %q",tc.tag,fn.dispatch.FnRender.Tag)}; if fn.dispatch.FnRender.TargetID!=tc.targetID {t.Fatalf("expected target id %q, got %q",tc.targetID,fn.dispatch.FnRender.TargetID)}; assertRenderMode(t,fn.dispatch.FnRender,renderRemove) }) }
}

func TestFnComponentClassMutations(t *testing.T) {
	add:=NewFn(context.Background(),nil).setClasses("status",false,"active","visible")
	assertMutation(t,add,"status","addClass",[]string{"active","visible"},"","")
	remove:=NewFn(context.Background(),nil).setClasses("status",true,"active")
	assertMutation(t,remove,"status","removeClass",[]string{"active"},"","")
}

func TestFnComponentDOMMutations(t *testing.T) {
	fn:=NewFn(context.Background(),nil).setDOM("email","setAttribute","aria-label","Email")
	assertMutation(t,fn,"email","setAttribute",nil,"aria-label","Email")
}

func assertMutation(t *testing.T, fn FnComponent, targetID, operation string, names []string, name, value string) {
	t.Helper()
	if fn.dispatch.Function!=mutate { t.Fatalf("expected mutate function, got %q",fn.dispatch.Function) }
	if fn.dispatch.FnMutate.TargetID!=targetID { t.Fatalf("expected target id %q, got %q",targetID,fn.dispatch.FnMutate.TargetID) }
	if len(fn.dispatch.FnMutate.Mutations)!=1 { t.Fatalf("expected one mutation, got %d",len(fn.dispatch.FnMutate.Mutations)) }
	m:=fn.dispatch.FnMutate.Mutations[0]
	if m.Operation!=operation || m.Name!=name || m.Value!=value { t.Fatalf("unexpected mutation: %+v",m) }
	if len(m.Names)!=len(names) { t.Fatalf("unexpected mutation names: %v",m.Names) }
	for i:=range names { if m.Names[i]!=names[i] { t.Fatalf("unexpected mutation names: %v",m.Names) } }
}

func assertRenderMode(t *testing.T, render FnRender, mode renderMode) {
	t.Helper()
	if render.Append!=(mode==renderAppend){t.Fatalf("append flag mismatch for mode %v",mode)}
	if render.Prepend!=(mode==renderPrepend){t.Fatalf("prepend flag mismatch for mode %v",mode)}
	if render.Inner!=(mode==renderInner){t.Fatalf("inner flag mismatch for mode %v",mode)}
	if render.Outer!=(mode==renderOuter){t.Fatalf("outer flag mismatch for mode %v",mode)}
	if render.Remove!=(mode==renderRemove){t.Fatalf("remove flag mismatch for mode %v",mode)}
}
