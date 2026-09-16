package neith

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type Component interface { Render(ctx context.Context, w io.Writer) error }

func RenderComponent(c ...Component) (html string) {
	w := Writer{}
	ctx := context.Background()
	for _, v := range c { _ = v.Render(ctx, &w) }
	return string(w.buf)
}

type FnComponent struct { context.Context; dispatch *Dispatch; id string }

type renderMode int
const ( renderAppend renderMode = iota; renderPrepend; renderInner; renderOuter; renderRemove )

func NewFn(ctx context.Context, c Component) FnComponent {
	id := "neith-" + uuid.New().String()
	dispatch := newDispatch(id)
	dispatch.rt = defaultRuntime
	dd, ok := dispatchFromContext(ctx)
	if !ok { defaultRuntime.Config().Logger.Warn(ErrCtxMissingDispatch) } else { dispatch.useContext(dd) }
	f := FnComponent{Context: ctx, id: id, dispatch: dispatch}.SwapTagInner("main")
	if c != nil { c.Render(f.Context, f) }
	return f
}

func (f FnComponent) Render(ctx context.Context, w io.Writer) error {
	if _, err := io.WriteString(w, f.openTag()); err != nil { return err }
	if err := HTML(f.dispatch.FnRender.HTML).Render(ctx, w); err != nil { return err }
	if _, err := w.Write(f.dispatch.buf); err != nil { return err }
	_, err := io.WriteString(w, "</div>"); return err
}
func (f FnComponent) Write(p []byte) (int, error) { f.dispatch.buf = append(f.dispatch.buf, p...); return len(p), nil }
func (f FnComponent) WithContext(ctx context.Context) FnComponent { f.Context=ctx; dd,ok:=dispatchFromContext(ctx); if !ok { f.dispatch.runtime().Config().Logger.Error(ErrCtxMissingDispatch); return f }; f.dispatch.useContext(dd); return f }
func (f FnComponent) WithEvents(h HandleFn, e ...OnEvent) FnComponent { for _,v:=range e { el:=newEventListener(v,f,h); if el.ID!="" { f.dispatch.FnRender.EventListeners=append(f.dispatch.FnRender.EventListeners,el) } }; return f }
func (f FnComponent) WithRedirect(url string) FnComponent { f.dispatch.Function=redirect; f.dispatch.FnRedirect.URL=url; return f }
func (f FnComponent) WithError(err error) FnComponent { if err==nil { err=errors.New("error is nil") }; f.dispatch.Function=fnError; f.dispatch.FnError.Message=err.Error(); return f }
func (f FnComponent) JS(fn string,arg any) FnComponent { f.dispatch.Function=custom; f.dispatch.FnCustom.Function=fn; f.dispatch.FnCustom.Data=arg; return f }
func (f FnComponent) WithLabel(label string) FnComponent { f.dispatch.Label=label; return f }
func (f FnComponent) openTag() string { events:=f.dispatch.FnRender.listenerStrings(); if f.dispatch.Label=="" { return fmt.Sprintf("<div id='%s' events=%s>",f.id,events) }; return fmt.Sprintf("<div id='%s' label='%s' events=%s>",f.id,f.dispatch.Label,events) }

func (f FnComponent) renderTag(tag string, mode renderMode) FnComponent { f.dispatch.Function=render; f.dispatch.FnRender.Tag=tag; f.dispatch.FnRender.TargetID=""; f.dispatch.FnRender.applyMode(mode); return f }
func (f FnComponent) renderElement(id string, mode renderMode) FnComponent { f.dispatch.Function=render; f.dispatch.FnRender.Tag=""; f.dispatch.FnRender.TargetID=id; f.dispatch.FnRender.applyMode(mode); return f }
func (f FnComponent) removeTag(tag string) FnComponent { return f.renderTag(tag,renderRemove) }
func (f FnComponent) removeElement(id string) FnComponent { return f.renderElement(id,renderRemove) }

// mutate configures one finite, allowlisted browser mutation. Existing public
// helpers route through this primitive so Phase 3 can simplify the wire without
// forcing a simultaneous public API migration.
func (f FnComponent) mutateElement(id string, mutation Mutation) FnComponent {
	f.dispatch.Function = mutate
	f.dispatch.FnMutate.TargetID = id
	f.dispatch.FnMutate.Mutations = []Mutation{mutation}
	return f
}
func (f FnComponent) setClasses(id string, remove bool, classes ...string) FnComponent {
	op := "addClass"; if remove { op="removeClass" }
	return f.mutateElement(id, Mutation{Operation:op, Names:classes})
}
func (f FnComponent) setDOM(id, operation, name, value string) FnComponent { return f.mutateElement(id, Mutation{Operation:operation, Name:name, Value:value}) }

func (f FnComponent) AppendTag(tag string) FnComponent { return f.renderTag(tag,renderAppend) }
func (f FnComponent) PrependTag(tag string) FnComponent { return f.renderTag(tag,renderPrepend) }
func (f FnComponent) SwapTagOuter(tag string) FnComponent { return f.renderTag(tag,renderOuter) }
func (f FnComponent) SwapTagInner(tag string) FnComponent { return f.renderTag(tag,renderInner) }
func (f FnComponent) AppendElement(id string) FnComponent { return f.renderElement(id,renderAppend) }
func (f FnComponent) PrependElement(id string) FnComponent { return f.renderElement(id,renderPrepend) }
func (f FnComponent) SwapElementOuter(id string) FnComponent { return f.renderElement(id,renderOuter) }
func (f FnComponent) SwapElementInner(id string) FnComponent { return f.renderElement(id,renderInner) }

func (f FnComponent) Dispatch() { if f.dispatch.conn==nil { f.dispatch.runtime().Config().Logger.Error(ErrConnectionNotFound); return }; h,ok:=f.dispatch.runtime().handlers.Get(f.dispatch.HandlerID); if !ok { f.dispatch.runtime().Config().Logger.Error("handler not found","HandlerID",f.dispatch.HandlerID); return }; h.out<-f }
func FnErr(ctx context.Context, err error) FnComponent { if err==nil { err=errors.New("error is nil") }; return NewFn(ctx,nil).WithError(err) }
func RedirectURL(ctx context.Context,url string) FnComponent { return NewFn(ctx,nil).WithRedirect(url) }
func JS(ctx context.Context,fn string,arg any) { NewFn(ctx,nil).JS(fn,arg).Dispatch() }
func AddClasses(ctx context.Context,id string,classes ...string) { NewFn(ctx,nil).setClasses(id,false,classes...).Dispatch() }
func RemoveClasses(ctx context.Context,id string,classes ...string) { NewFn(ctx,nil).setClasses(id,true,classes...).Dispatch() }
func SetAttribute(ctx context.Context,id,name,value string) { NewFn(ctx,nil).setDOM(id,"setAttribute",name,value).Dispatch() }
func RemoveAttribute(ctx context.Context,id,name string) { NewFn(ctx,nil).setDOM(id,"removeAttribute",name,"").Dispatch() }
func SetStyle(ctx context.Context,id,name,value string) { NewFn(ctx,nil).setDOM(id,"setStyle",name,value).Dispatch() }
func RemoveStyle(ctx context.Context,id,name string) { NewFn(ctx,nil).setDOM(id,"removeStyle",name,"").Dispatch() }
func SetText(ctx context.Context,id,value string) { NewFn(ctx,nil).setDOM(id,"setText","",value).Dispatch() }
func SetValue(ctx context.Context,id,value string) { NewFn(ctx,nil).setDOM(id,"setValue","",value).Dispatch() }
func Focus(ctx context.Context,id string) { NewFn(ctx,nil).setDOM(id,"focus","","").Dispatch() }
func Blur(ctx context.Context,id string) { NewFn(ctx,nil).setDOM(id,"blur","","").Dispatch() }
func ScrollIntoView(ctx context.Context,id string) { NewFn(ctx,nil).setDOM(id,"scrollIntoView","","").Dispatch() }
func Disable(ctx context.Context,id string) { NewFn(ctx,nil).setDOM(id,"disable","","").Dispatch() }
func Enable(ctx context.Context,id string) { NewFn(ctx,nil).setDOM(id,"enable","","").Dispatch() }
func RemoveElement(ctx context.Context,id string) { NewFn(ctx,nil).removeElement(id).Dispatch() }
func RemoveTag(ctx context.Context,tag string) { NewFn(ctx,nil).removeTag(tag).Dispatch() }

type HTML string
func (h HTML) Render(ctx context.Context,w io.Writer) error { _,err:=w.Write([]byte(h)); return err }
func (h *HTML) Write(p []byte)(int,error){ *h=HTML(string(*h)+string(p)); return len(p),nil }
func (d *Dispatch) useContext(details dispatchDetails){ d.rt=runtimeFromDispatch(details); d.ConnID=details.ClientID; d.HandlerID=details.HandlerID; d.conn=details.Conn }
func (d *Dispatch) runtime()*runtime{ if d==nil||d.rt==nil{return defaultRuntime}; return d.rt }
func (r *FnRender) applyMode(mode renderMode){ r.Append=mode==renderAppend; r.Prepend=mode==renderPrepend; r.Inner=mode==renderInner; r.Outer=mode==renderOuter; r.Remove=mode==renderRemove }
