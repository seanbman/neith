package fcmp

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// Component is the minimal renderable unit fcmp can send to the browser.
//
// It intentionally matches the shape used by templ and many simple Go HTML
// renderers: Render receives a context and writes HTML to the supplied writer.
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

// RenderComponent renders one or more components into a single HTML string.
//
// This helper is useful when code needs a component's HTML outside the live
// websocket dispatch path. It renders with context.Background and appends each
// component into the same buffer in the order provided.
func RenderComponent(c ...Component) (html string) {
	w := Writer{}
	ctx := context.Background()
	for _, v := range c {
		_ = v.Render(ctx, &w)
	}
	html = string(w.buf)
	return html
}

// FnComponent is the server-side wrapper around rendered HTML and dispatch data.
//
// It implements io.Writer so components can render directly into it. The
// dispatch field tracks how the browser should apply the rendered HTML, which
// websocket connection it belongs to, and any event listeners attached to it.
type FnComponent struct {
	context.Context
	dispatch *Dispatch
	id       string
}

// renderMode describes how rendered HTML should be applied in the DOM.
//
// The browser client expects a group of boolean flags on FnRender. Keeping the
// mode as one local enum lets the public methods read like intent while
// applyMode is the only place that translates the mode into those flags.
type renderMode int

const (
	renderAppend renderMode = iota
	renderPrepend
	renderInner
	renderOuter
	renderRemove
)

// NewFn creates a new FnComponent from a Component.
//
// The component is given a unique DOM wrapper ID and, when ctx was created by
// fcmp middleware, is attached to the active websocket dispatch context. NewFn
// defaults to swapping the inner HTML of <main>, which gives simple apps a
// useful render target without extra setup.
func NewFn(ctx context.Context, c Component) FnComponent {
	id := "fcmp-" + uuid.New().String()

	dispatch := newDispatch(id)
	dd, ok := dispatchFromContext(ctx)
	if !ok {
		config.Logger.Warn(ErrCtxMissingDispatch)
	} else {
		dispatch.useContext(dd)
	}

	f := FnComponent{
		Context:  ctx,
		id:       id,
		dispatch: dispatch,
	}.SwapTagInner("main")
	if c != nil {
		c.Render(f.Context, f)
	}
	return f
}

// Render writes the component wrapper and buffered HTML.
//
// The wrapper div carries the component ID, optional label, and serialized event
// listener metadata. The browser client reads those attributes after inserting
// the HTML so it can attach DOM listeners and route events back to Go.
func (f FnComponent) Render(ctx context.Context, w io.Writer) error {
	if _, err := io.WriteString(w, f.openTag()); err != nil {
		return err
	}
	if err := HTML(f.dispatch.FnRender.HTML).Render(ctx, w); err != nil {
		return err
	}
	if _, err := w.Write(f.dispatch.buf); err != nil {
		return err
	}
	_, err := io.WriteString(w, "</div>")
	return err
}

// Write appends rendered bytes to this component's dispatch buffer.
//
// Component renderers call this indirectly when NewFn renders a Component into
// the FnComponent. The buffered bytes are later wrapped by Render and sent to
// the browser.
func (f FnComponent) Write(p []byte) (n int, err error) {
	f.dispatch.buf = append(f.dispatch.buf, p...)
	return len(p), nil
}

// WithContext replaces the component context and refreshes dispatch details.
//
// Use this when a component value is created outside the request/event context
// and later needs to be associated with the active connection before dispatch.
func (f FnComponent) WithContext(ctx context.Context) FnComponent {
	f.Context = ctx

	dd, ok := dispatchFromContext(ctx)
	if !ok {
		config.Logger.Error(ErrCtxMissingDispatch)
		return f
	}
	f.dispatch.useContext(dd)
	return f
}

// WithEvents attaches one server handler to one or more DOM event types.
//
// Each event listener is registered in the connection-local event registry and
// serialized into the component's wrapper metadata. When the browser receives
// this component, it attaches listeners and sends matching DOM events back to h.
func (f FnComponent) WithEvents(h HandleFn, e ...OnEvent) FnComponent {
	for _, v := range e {
		el := newEventListener(v, f, h)
		f.dispatch.FnRender.EventListeners = append(f.dispatch.FnRender.EventListeners, el)
	}
	return f
}

// WithRedirect changes this dispatch into a browser redirect.
//
// Redirect components do not need rendered HTML. When returned from a handler or
// dispatched directly, the browser client sets window.location to url.
func (f FnComponent) WithRedirect(url string) FnComponent {
	f.dispatch.Function = redirect
	f.dispatch.FnRedirect.URL = url
	return f
}

// WithError changes this dispatch into an error message.
//
// The server-side handler pipeline logs these errors through the package logger
// unless logging is disabled. A nil error is normalized to a useful message so
// callers do not silently dispatch an empty error.
func (f FnComponent) WithError(err error) FnComponent {
	if err == nil {
		err = errors.New("error is nil")
	}
	f.dispatch.Function = _error
	f.dispatch.FnError.Message = err.Error()
	return f
}

// JS changes this dispatch into a custom browser-side JavaScript call.
//
// fn is the name of a function on window and arg is serialized as the payload.
// The browser client calls window[fn](arg) and sends the result back in the
// dispatch's custom result field.
func (f FnComponent) JS(fn string, arg any) FnComponent {
	f.dispatch.Function = custom
	f.dispatch.FnCustom.Function = fn
	f.dispatch.FnCustom.Data = arg
	return f
}

// WithLabel sets a human-readable label on the component wrapper.
//
// The label may be used to identify a component on the server and client,
// especially during debugging.
func (f FnComponent) WithLabel(label string) FnComponent {
	f.dispatch.Label = label
	return f
}

// openTag builds the wrapper div's opening tag.
//
// Render uses this so label handling and event-listener serialization stay in
// one place. The events attribute contains JSON that the browser client parses
// after inserting the rendered component.
func (f FnComponent) openTag() string {
	events := f.dispatch.FnRender.listenerStrings()
	if f.dispatch.Label == "" {
		return fmt.Sprintf("<div id='%s' events=%s>", f.id, events)
	}
	return fmt.Sprintf("<div id='%s' label='%s' events=%s>", f.id, f.dispatch.Label, events)
}

// renderTag targets the first matching DOM tag and sets the render mode.
//
// This is the shared implementation behind AppendTag, PrependTag, and the tag
// swap helpers. It clears TargetID so the browser client uses the tag selector.
func (f FnComponent) renderTag(tag string, mode renderMode) FnComponent {
	f.dispatch.Function = render
	f.dispatch.FnRender.Tag = tag
	f.dispatch.FnRender.TargetID = ""
	f.dispatch.FnRender.applyMode(mode)
	return f
}

// renderElement targets one DOM element by ID and sets the render mode.
//
// This is the shared implementation behind AppendElement, PrependElement, and
// the element swap helpers. It clears Tag so the browser client uses TargetID.
func (f FnComponent) renderElement(id string, mode renderMode) FnComponent {
	f.dispatch.Function = render
	f.dispatch.FnRender.Tag = ""
	f.dispatch.FnRender.TargetID = id
	f.dispatch.FnRender.applyMode(mode)
	return f
}

// removeTag configures this component to remove the first matching DOM tag.
//
// It shares the same render dispatch function as normal renders, but sets the
// render mode to remove and leaves HTML empty.
func (f FnComponent) removeTag(tag string) FnComponent {
	return f.renderTag(tag, renderRemove)
}

// removeElement configures this component to remove one DOM element by ID.
//
// It clears the default tag target inherited from NewFn, which keeps element
// removal unambiguous for the browser client.
func (f FnComponent) removeElement(id string) FnComponent {
	return f.renderElement(id, renderRemove)
}

// setClasses configures this component to add or remove CSS classes.
//
// The remove flag selects between classList.add and classList.remove on the
// browser client. The classes slice is forwarded as-is to preserve call order.
func (f FnComponent) setClasses(id string, remove bool, classes ...string) FnComponent {
	f.dispatch.Function = class
	f.dispatch.FnClass.TargetID = id
	f.dispatch.FnClass.Remove = remove
	f.dispatch.FnClass.Names = classes
	return f
}

// AppendTag appends the rendered component to the first matching tag in the DOM.
//
// The browser client finds document.getElementsByTagName(tag)[0] and appends the
// rendered HTML to that element's innerHTML.
func (f FnComponent) AppendTag(tag string) FnComponent {
	return f.renderTag(tag, renderAppend)
}

// PrependTag prepends the rendered component to the first matching tag.
//
// The browser client inserts the rendered HTML before the existing innerHTML of
// document.getElementsByTagName(tag)[0].
func (f FnComponent) PrependTag(tag string) FnComponent {
	return f.renderTag(tag, renderPrepend)
}

// SwapTagOuter replaces the first matching tag's outer HTML.
//
// Use this when the rendered component should replace the target element itself,
// not just the target element's contents.
func (f FnComponent) SwapTagOuter(tag string) FnComponent {
	return f.renderTag(tag, renderOuter)
}

// SwapTagInner replaces the first matching tag's inner HTML.
//
// This is NewFn's default render mode for <main>, making it the common path for
// full-page or main-region updates.
func (f FnComponent) SwapTagInner(tag string) FnComponent {
	return f.renderTag(tag, renderInner)
}

// AppendElement appends the rendered component to an element by ID.
//
// The browser client finds document.getElementById(id) and appends the rendered
// HTML to that element's innerHTML.
func (f FnComponent) AppendElement(id string) FnComponent {
	return f.renderElement(id, renderAppend)
}

// PrependElement prepends the rendered component to an element by ID.
//
// The browser client finds document.getElementById(id) and inserts the rendered
// HTML before the element's existing innerHTML.
func (f FnComponent) PrependElement(id string) FnComponent {
	return f.renderElement(id, renderPrepend)
}

// SwapElementOuter replaces one element's outer HTML by ID.
//
// Use this when the rendered component should replace the selected element
// itself, including its tag.
func (f FnComponent) SwapElementOuter(id string) FnComponent {
	return f.renderElement(id, renderOuter)
}

// SwapElementInner replaces one element's inner HTML by ID.
//
// Use this when the target element should remain in place but its contents
// should be replaced by the rendered component.
func (f FnComponent) SwapElementInner(id string) FnComponent {
	return f.renderElement(id, renderInner)
}

// Dispatch immediately queues this component for the active browser connection.
//
// Dispatch requires a connection and handler ID from middleware context. If the
// component was created outside an fcmp request/event context, Dispatch logs the
// missing connection and returns without sending anything.
func (f FnComponent) Dispatch() {
	if f.dispatch.conn == nil {
		config.Logger.Error(ErrConnectionNotFound)
		return
	}
	h, ok := handlers.Get(f.dispatch.HandlerID)
	if !ok {
		config.Logger.Error("handler not found", "HandlerID", f.dispatch.HandlerID)
		return
	}
	h.out <- f
}

// FnErr creates an error dispatch from a context and error.
//
// This is a convenience for event handlers that return FnComponent and want to
// surface an error through fcmp's normal error path.
func FnErr(ctx context.Context, err error) FnComponent {
	if err == nil {
		err = errors.New("error is nil")
	}
	return NewFn(ctx, nil).WithError(err)
}

// RedirectURL creates a redirect dispatch for handler return values.
//
// Returning this from a HandleFn tells the browser client to navigate to url.
func RedirectURL(ctx context.Context, url string) FnComponent {
	return NewFn(ctx, nil).WithRedirect(url)
}

// JS dispatches a custom JavaScript call immediately.
//
// Unlike FnComponent.JS, this helper sends the dispatch right away and does not
// return a component for a handler to return.
func JS(ctx context.Context, fn string, arg any) {
	NewFn(ctx, nil).JS(fn, arg).Dispatch()
}

// AddClasses immediately adds CSS classes to one element by ID.
//
// The mutation is sent as a class dispatch and runs through classList.add on the
// browser client.
func AddClasses(ctx context.Context, id string, classes ...string) {
	NewFn(ctx, nil).setClasses(id, false, classes...).Dispatch()
}

// RemoveClasses immediately removes CSS classes from one element by ID.
//
// The mutation is sent as a class dispatch and runs through classList.remove on
// the browser client.
func RemoveClasses(ctx context.Context, id string, classes ...string) {
	NewFn(ctx, nil).setClasses(id, true, classes...).Dispatch()
}

// RemoveElement immediately removes one DOM element by ID.
//
// This sends a render dispatch with Remove set and TargetID populated.
func RemoveElement(ctx context.Context, id string) {
	NewFn(ctx, nil).removeElement(id).Dispatch()
}

// RemoveTag immediately removes the first matching DOM tag.
//
// This sends a render dispatch with Remove set and Tag populated.
func RemoveTag(ctx context.Context, tag string) {
	NewFn(ctx, nil).removeTag(tag).Dispatch()
}

// HTML adapts a raw HTML string to the Component interface.
//
// It is useful for small examples, tests, and simple components that do not need
// a dedicated struct or templ-generated renderer.
type HTML string

// Render writes the HTML string to the provided writer.
//
// The context parameter is accepted to satisfy Component; HTML itself does not
// inspect it.
func (h HTML) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(h))
	return err
}

// Write appends bytes to the HTML value.
//
// This lets HTML act as a small string buffer when code wants an io.Writer that
// accumulates rendered HTML.
func (h *HTML) Write(p []byte) (n int, err error) {
	*h = HTML(string(*h) + string(p))
	return len(p), nil
}

// useContext copies websocket dispatch details onto a Dispatch.
//
// NewFn and WithContext both use this to keep connection ID, handler ID, and
// connection pointer assignment consistent.
func (d *Dispatch) useContext(details dispatchDetails) {
	d.ConnID = details.ConnID
	d.HandlerID = details.HandlerID
	d.conn = details.Conn
}

// applyMode translates a renderMode into the boolean flags sent to the client.
//
// The browser protocol currently represents render behavior as booleans. This
// helper guarantees only the selected mode flag is true.
func (r *FnRender) applyMode(mode renderMode) {
	r.Append = mode == renderAppend
	r.Prepend = mode == renderPrepend
	r.Inner = mode == renderInner
	r.Outer = mode == renderOuter
	r.Remove = mode == renderRemove
}
