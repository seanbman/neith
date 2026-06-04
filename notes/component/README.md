# Component

The component API is the core of fcmp's rendering model. A component renders
HTML on the server, and `FnComponent` carries that HTML plus instructions for
how the browser should apply it.

## Basic Flow

```go
func app(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`
		<button>Click me</button>
	`)).WithEvents(clicked, fcmp.OnClick)
}

func clicked(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`
		<section>
			<h1>Hello</h1>
			<p>This was rendered on the server.</p>
		</section>
	`)).SwapTagInner("main")
}
```

## `Component`

Any type that implements this interface can be rendered by fcmp:

```go
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}
```

Example custom component:

```go
type Greeting struct {
	Name string
}

func (g Greeting) Render(ctx context.Context, w io.Writer) error {
	_, err := fmt.Fprintf(w, "<h1>Hello %s</h1>", g.Name)
	return err
}
```

Usage:

```go
return fcmp.NewFn(ctx, Greeting{Name: "Sean"})
```

## `RenderComponent(components...)`

Renders one or more components into a plain HTML string.

```go
html := fcmp.RenderComponent(
	fcmp.HTML(`<h1>Title</h1>`),
	fcmp.HTML(`<p>Body</p>`),
)
```

Notes:

- This does not send anything to a browser.
- It does not require a websocket connection.
- It is useful for tests, logging, server-side composition, or initial HTML.

## `HTML`

Adapts a raw HTML string to the `Component` interface.

```go
component := fcmp.HTML(`<button>Save</button>`)
return fcmp.NewFn(ctx, component)
```

Notes:

- Best for examples and simple fragments.
- For larger UIs, use your own `Component` type or a renderer like templ.

## `NewFn(ctx, component)`

Creates an `FnComponent` from a renderable component.

```go
fn := fcmp.NewFn(ctx, fcmp.HTML(`<h1>Dashboard</h1>`))
```

Notes:

- Gives the component a unique wrapper ID.
- Pulls websocket dispatch details from `ctx` when available.
- Renders the supplied component into an internal buffer.
- Defaults to replacing the inner HTML of the first `<main>` tag.

Common usage:

```go
return fcmp.NewFn(ctx, fcmp.HTML(`<p>Updated</p>`))
```

## `FnComponent.Render(ctx, writer)`

Writes the component wrapper and buffered HTML.

```go
var out strings.Builder
err := fcmp.NewFn(ctx, fcmp.HTML(`<p>Hello</p>`)).Render(ctx, &out)
```

Notes:

- Usually called internally by fcmp.
- The wrapper contains event metadata for the browser client.
- Returns writer errors.

## `FnComponent.Write(bytes)`

Appends bytes to the component's internal render buffer.

```go
fn := fcmp.NewFn(ctx, nil)
_, _ = fn.Write([]byte(`<p>Manual HTML</p>`))
```

Notes:

- Component renderers use this when rendering into `FnComponent`.
- Most callers do not need to call it directly.

## `WithContext(ctx)`

Replaces the component context and refreshes connection dispatch details.

```go
fn = fn.WithContext(ctx)
```

Use this when a component was created before the active fcmp context was
available, but needs to be dispatched later.

## `WithEvents(handler, events...)`

Attaches server-side handlers to browser DOM events.

```go
return fcmp.NewFn(ctx, fcmp.HTML(`<button>Save</button>`)).
	WithEvents(save, fcmp.OnClick)
```

Multiple events:

```go
return fcmp.NewFn(ctx, input).
	WithEvents(update, fcmp.OnInput, fcmp.OnChange)
```

Notes:

- Event metadata is serialized into the rendered HTML.
- The browser client attaches DOM listeners.
- When the event fires, the browser sends data back to the server.

## `WithRedirect(url)`

Turns the component into a redirect dispatch.

```go
return fcmp.NewFn(ctx, nil).WithRedirect("/dashboard")
```

Shortcut:

```go
return fcmp.RedirectURL(ctx, "/dashboard")
```

## `WithError(err)`

Turns the component into an error dispatch.

```go
return fcmp.NewFn(ctx, nil).WithError(err)
```

Shortcut:

```go
return fcmp.FnErr(ctx, err)
```

Notes:

- A nil error becomes `"error is nil"`.
- Errors are logged by the server handler path unless logging is disabled.

## `JS(fn, arg)`

Configures a component dispatch to call a browser-side JavaScript function.

```go
return fcmp.NewFn(ctx, nil).JS("showToast", "Saved")
```

Immediate shortcut:

```go
fcmp.JS(ctx, "showToast", "Saved")
```

Browser-side expectation:

```js
window.showToast = function (message) {
	console.log(message)
	return true
}
```

## `WithLabel(label)`

Adds a label attribute to the rendered wrapper.

```go
return fcmp.NewFn(ctx, component).WithLabel("counter")
```

Notes:

- Useful for debugging rendered components.
- The label is sent to the browser as metadata.

## Render Target Methods

These methods all configure `FnRender`, then return the component so calls can
be chained.

### `AppendTag(tag)`

Appends rendered HTML to the first matching tag.

```go
return fcmp.NewFn(ctx, item).AppendTag("ul")
```

### `PrependTag(tag)`

Prepends rendered HTML to the first matching tag.

```go
return fcmp.NewFn(ctx, alert).PrependTag("main")
```

### `SwapTagOuter(tag)`

Replaces the first matching tag itself.

```go
return fcmp.NewFn(ctx, page).SwapTagOuter("main")
```

### `SwapTagInner(tag)`

Replaces the contents of the first matching tag.

```go
return fcmp.NewFn(ctx, page).SwapTagInner("main")
```

This is the default mode used by `NewFn`.

### `AppendElement(id)`

Appends rendered HTML to an element by ID.

```go
return fcmp.NewFn(ctx, item).AppendElement("items")
```

### `PrependElement(id)`

Prepends rendered HTML to an element by ID.

```go
return fcmp.NewFn(ctx, notice).PrependElement("messages")
```

### `SwapElementOuter(id)`

Replaces the selected element itself.

```go
return fcmp.NewFn(ctx, card).SwapElementOuter("profile-card")
```

### `SwapElementInner(id)`

Replaces the selected element's contents.

```go
return fcmp.NewFn(ctx, form).SwapElementInner("settings")
```

## `Dispatch()`

Immediately sends the configured component to the connected browser.

```go
fcmp.NewFn(ctx, fcmp.HTML(`<p>Saved</p>`)).
	SwapElementInner("status").
	Dispatch()
```

Notes:

- Requires a valid fcmp context with a websocket connection.
- Event handlers usually return `FnComponent` instead of calling `Dispatch`.
- Use `Dispatch` for side effects inside a handler.

## Helper Functions

### `FnErr(ctx, err)`

Creates an error component.

```go
if err != nil {
	return fcmp.FnErr(ctx, err)
}
```

### `RedirectURL(ctx, url)`

Creates a redirect component.

```go
return fcmp.RedirectURL(ctx, "/login")
```

### `JS(ctx, fn, arg)`

Immediately calls a browser-side JavaScript function.

```go
fcmp.JS(ctx, "showToast", "Saved")
```

### `AddClasses(ctx, id, classes...)`

Immediately adds CSS classes to an element by ID.

```go
fcmp.AddClasses(ctx, "status", "is-active", "text-green")
```

### `RemoveClasses(ctx, id, classes...)`

Immediately removes CSS classes from an element by ID.

```go
fcmp.RemoveClasses(ctx, "status", "is-active")
```

### `RemoveElement(ctx, id)`

Immediately removes an element by ID.

```go
fcmp.RemoveElement(ctx, "modal")
```

### `RemoveTag(ctx, tag)`

Immediately removes the first matching tag.

```go
fcmp.RemoveTag(ctx, "dialog")
```

## Internal Flow

```text
NewFn(ctx, component)
        |
        v
create Dispatch + unique component ID
        |
        v
copy websocket context details when available
        |
        v
render Component into FnComponent buffer
        |
        v
configure render mode / events / redirect / class / custom action
        |
        v
return from handler or call Dispatch()
        |
        v
handler publishes JSON Dispatch to browser
```
