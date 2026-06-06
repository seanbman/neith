<p align="center">
  <img src="static/logo.png" alt="Neith logo" width="280">
</p>

# Neith

Neith is a small Go package for building server-rendered, interactive HTML components.
It wraps Go components with a WebSocket-backed dispatch layer so the server can render
HTML, respond to DOM events, update elements, redirect the browser, call client-side
JavaScript, and keep per-client state.

The package is designed around a minimal component interface:

```go
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}
```

That means plain `neith.HTML`, `templ` components, or your own renderable types can be
used as Neith components.

## Table Of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Browser Assets](#browser-assets)
- [Rendering Components](#rendering-components)
- [Readable Views And UI Components](#readable-views-and-ui-components)
- [Events](#events)
  - [File Uploads](#file-uploads)
- [Server-Initiated Updates](#server-initiated-updates)
- [Client Sessions And Cache](#client-sessions-and-cache)
- [Configuration](#configuration)
- [API Examples](#api-examples)
  - [App And Page](#app-and-page)
  - [View Options](#view-options)
  - [Components And Dispatch](#components-and-dispatch)
  - [Event Data](#event-data)
  - [DOM Effects](#dom-effects)
  - [Cache](#cache)
  - [Configuration Helpers](#configuration-helpers)
  - [UI Components](#ui-components)
  - [UI Options](#ui-options)
- [Example App](#example-app)
- [Detailed Notes](#detailed-notes)
- [Development](#development)
- [License](#license)

## Features

- Render Go components into the DOM over WebSockets.
- Attach server-side handlers to browser events.
- Swap, append, prepend, or remove elements by tag or ID.
- Add and remove CSS classes from the server.
- Redirect the browser from a handler.
- Run custom JavaScript functions from Go.
- Read typed event payloads with `EventData[T]`.
- Inspect uploaded files and form submitter metadata from handlers.
- Store per-client-session state with generic caches.
- Configure logging and cache expiry.

## Installation

```sh
go get github.com/seanbman/neith
```

Neith requires Go 1.21 or newer.

## Quick Start

Mount your app with `neith.App`. It serves the default page, the browser client,
and the neutral `ui` stylesheet for you.

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/seanbman/neith"
)

func app(ctx context.Context) neith.FnComponent {
	return neith.View(ctx, neith.HTML(`
		<button>Click me</button>
	`), neith.OnClick(clicked))
}

func clicked(ctx context.Context) neith.FnComponent {
	return neith.View(ctx, neith.HTML(`
		<section>
			<h1>Hello from the server</h1>
			<p>The button click was handled in Go.</p>
		</section>
	`), neith.IntoTag("main"))
}

func main() {
	http.HandleFunc("/", neith.App(app, neith.Title("Neith demo")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Open `http://localhost:8080`. The first request serves Neith's default page. The
browser client then opens a WebSocket back to the same route with `?neith_id=...`,
and `App` sends the initial `FnComponent` to the client.

Customize the page when you need to, without hand-writing the shell:

```go
http.HandleFunc("/", neith.App(app,
	neith.Title("Admin console"),
	neith.Target("main", "app"),
	neith.Head(neith.HTML(`<meta name="theme-color" content="#172026">`)),
))
```

Each call to `App` creates an isolated Neith app runtime for that route. The
browser's `neith_id` identifies one client session inside that app runtime, while
the WebSocket is only the current live transport for the session. Refreshes or
reconnects replace the transport without sharing cache or event state with other
mounted Neith routes.

## Browser Assets

`neith.App` serves the browser client for you from the embedded asset at
`/assets/neith.min.js`. Most apps do not need to download or copy the file.

If you use `MiddleWareFn` with a custom page or want to self-host the asset,
download:

```text
https://raw.githubusercontent.com/seanbman/neith/main/static/assets/neith.min.js
```

Save it in your app at:

```text
static/assets/neith.min.js
```

Then reference it from your page:

```html
<script defer src="/assets/neith.min.js"></script>
```

The optional neutral stylesheet is available at:

```text
https://raw.githubusercontent.com/seanbman/neith/main/static/assets/neith-ui.css
```

Save it as:

```text
static/assets/neith-ui.css
```

For release-pinned apps, replace `main` in the download URLs with the tag or
branch you want to vendor.

## Rendering Components

Create an `FnComponent` with `NewFn(ctx, component)`. By default, it swaps the inner
HTML of the first `<main>` tag.

```go
func view(ctx context.Context) neith.FnComponent {
	return neith.NewFn(ctx, neith.HTML(`<h1>Dashboard</h1>`))
}
```

You can change where the rendered HTML is applied:

```go
neith.NewFn(ctx, component).SwapTagInner("main")
neith.NewFn(ctx, component).SwapTagOuter("main")
neith.NewFn(ctx, component).AppendTag("ul")
neith.NewFn(ctx, component).PrependTag("ul")

neith.NewFn(ctx, component).SwapElementInner("content")
neith.NewFn(ctx, component).SwapElementOuter("content")
neith.NewFn(ctx, component).AppendElement("items")
neith.NewFn(ctx, component).PrependElement("items")
```

To render components to a string outside the live dispatch flow:

```go
html := neith.RenderComponent(neith.HTML(`<p>Rendered</p>`))
```

## Readable Views And UI Components

`neith.View` is a readability-focused wrapper around `NewFn`. It accepts the same
minimal `neith.Component` interface, so existing `templ` components, raw
`neith.HTML`, and custom Go renderers can be moved into Neith without changing how
they render.

```go
func app(ctx context.Context) neith.FnComponent {
	return neith.View(ctx, dashboard(rows),
		neith.Label("dashboard"),
		neith.OnSubmit(save),
		neith.IntoTag("main"),
	)
}
```

The lower-level API is still available when you want to step through every
operation directly:

```go
return neith.NewFn(ctx, dashboard(rows)).
	WithLabel("dashboard").
	WithEvents(save, neith.EventSubmit).
	SwapTagInner("main")
```

Neith also includes an optional `ui` package for small, generic application
components. These components are normal `neith.Component` values, so they can be
mixed with `templ` and raw HTML.

```go
import "github.com/seanbman/neith/ui"

func settings(ctx context.Context) neith.FnComponent {
	return neith.View(ctx,
		ui.Panel(
			ui.Heading("Settings", ui.Level(2)),
			profileForm(), // existing templ component
			neith.HTML(`<hr>`),
			ui.Form(
				ui.HiddenInput("intent", "save"),
				ui.TextInput("source",
					ui.Label("Source"),
					ui.Value("Billing service"),
				),
				ui.Select("status",
					ui.Label("Status"),
					ui.Options("ok", "queued", "warning"),
				),
				ui.TextArea("message",
					ui.Label("Message"),
					ui.Value("Invoice reconciliation completed"),
				),
				ui.Button("Save", ui.Type("submit"), ui.Primary()),
			),
		),
		neith.Label("settings"),
		neith.OnSubmit(saveSettings),
		neith.IntoElement("content"),
	)
}
```

Use `View` to attach behavior and render targets. Use `ui` only where generic
components make application code easier to read. Existing `templ` files can remain
the source of detailed markup.

The optional `/assets/neith-ui.css` stylesheet gives `ui` components neutral
defaults and exposes CSS variables such as `--n-ui-primary-bg`,
`--n-ui-border`, `--n-ui-radius`, and `--n-ui-font` for application themes.

## Events

Attach one or more DOM events with `WithEvents`.

```go
func form(ctx context.Context) neith.FnComponent {
	return neith.NewFn(ctx, neith.HTML(`
		<form>
			<input name="name" placeholder="Name">
			<button>Save</button>
		</form>
	`)).WithEvents(save, neith.EventSubmit)
}

func save(ctx context.Context) neith.FnComponent {
	values, err := neith.EventData[map[string]string](ctx)
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	return neith.NewFn(ctx, neith.HTML(
		"<p>Saved " + values["name"] + "</p>",
	)).SwapTagInner("main")
}
```

For pointer, mouse, keyboard, drag, touch, input, change, submit, and other DOM events,
the browser client sends event data back to Go. Use `EventData[T]` with the matching
type, such as `neith.PointerEvent`, `neith.DragEvent`, or your own form-data struct/map.

Event targets include the element ID, name, classes, tag name, HTML, value, checked,
disabled, hidden, inline style, attributes, dataset, and selected option values. For
mouse, pointer, drag, touch, and keyboard payloads, `source` is the element that
caused the event and `component` is the Neith wrapper listening for it. For submit
events, `EventSubmitter` returns the button or input that submitted the form.

```go
func save(ctx context.Context) neith.FnComponent {
	values, err := neith.EventData[map[string]string](ctx)
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	submitter, err := neith.EventSubmitter(ctx)
	if err != nil {
		return neith.FnErr(ctx, err)
	}
	if submitter != nil && submitter.Value == "delete" {
		return neith.NewFn(ctx, neith.HTML("<p>Delete requested</p>"))
	}

	return neith.NewFn(ctx, neith.HTML("<p>Saved " + values["name"] + "</p>"))
}
```

### File Uploads

Forms with file inputs upload file bytes over HTTP before the websocket event is
sent. Normal form values stay available through `EventData`, and uploaded file
metadata is available with `EventUploads`.

```go
func upload(ctx context.Context) neith.FnComponent {
	values, err := neith.EventData[map[string]string](ctx)
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	uploads, err := neith.EventUploads(ctx)
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	return neith.NewFn(ctx, neith.HTML(
		"<p>Saved " + values["title"] + " with " + uploads[0].FileName + "</p>",
	))
}
```

Uploaded files are written to `Config.UploadDir`, or to the system temp
directory under `neith-uploads` when no directory is configured.

## Server-Initiated Updates

Handlers can dispatch extra effects while handling an event:

```go
func clicked(ctx context.Context) neith.FnComponent {
	neith.AddClasses(ctx, "status", "active")
	neith.RemoveClasses(ctx, "status", "pending")
	neith.JS(ctx, "Testing", "called from Go")

	return neith.NewFn(ctx, neith.HTML(`<p id="status">Done</p>`))
}
```

Other helpers:

```go
neith.SetAttribute(ctx, "save", "aria-busy", "true")
neith.RemoveAttribute(ctx, "save", "aria-busy")
neith.SetStyle(ctx, "status", "color", "green")
neith.RemoveStyle(ctx, "status", "color")
neith.SetText(ctx, "status", "Saved")
neith.SetValue(ctx, "search", "")
neith.Focus(ctx, "search")
neith.Blur(ctx, "search")
neith.ScrollIntoView(ctx, "results")
neith.Disable(ctx, "save")
neith.Enable(ctx, "save")
neith.RemoveElement(ctx, "modal")
neith.RemoveTag(ctx, "dialog")
return neith.RedirectURL(ctx, "/next")
```

The browser client also exposes lifecycle hooks:

```js
window.neith.on("afterRender", ({ dispatch, element }) => {
	console.log("rendered", dispatch.function, element)
})

window.neith.on("beforeEventDispatch", ({ dispatch, event }) => {
	console.log("sending event", dispatch.event.on)
})
```

The socket emits `connect`, `disconnect`, and `reconnect` hooks. Unexpected
disconnects are retried with capped exponential backoff instead of reloading the page.

## Client Sessions And Cache

Neith includes a generic cache scoped to the current client session in the current
app runtime. Create a cache once, then reuse it from later handlers for the same
browser client. Two visitors can use the same cache keys without sharing values,
and two routes wrapped with `MiddleWareFn` keep separate runtimes.

```go
func app(ctx context.Context) neith.FnComponent {
	_, _ = neith.NewCache(ctx, "count", 0)
	return counter(ctx)
}

func counter(ctx context.Context) neith.FnComponent {
	count, err := neith.UseCache[int](ctx, "count")
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	_ = count.Set(count.Value() + 1)

	return neith.NewFn(ctx, neith.HTML(fmt.Sprintf(
		`<button>Clicked %d times</button>`,
		count.Value(),
	))).WithEvents(func(ctx context.Context) neith.FnComponent {
		return counter(ctx)
	}, neith.EventClick)
}
```

Cache helpers include:

- `Set(value, timeout...)`
- `Value()`
- `Delete()`
- `CreatedAt()`
- `UpdatedAt()`
- `Expiry()`
- `Record(true)`
- `History()`
- `OnCacheChange(cache, fn)`
- `OnCacheTimeOut(cache, fn)`

Use `Record(true)` before calling `Set` when you want to keep a history of cache
updates:

```go
cache, err := neith.UseCache[int](ctx, "count")
if err != nil {
	return neith.FnErr(ctx, err)
}

cache.Record(true)
_ = cache.Set(cache.Value() + 1)

history, ok := cache.History()
if ok {
	// history contains recorded values keyed by update time.
}
```

## Configuration

The default config uses a 30 minute cache timeout and logs errors.

```go
neith.SetConfig(&neith.Config{
	CacheTimeOut: 10 * time.Minute,
	LogLevel:     neith.Info,
})
```

Set `Silent: true` or `LogLevel: neith.None` to disable package logs.

## API Examples

These examples cover the client-facing functions in `neith` and `neith/ui`.
Protocol payload structs such as `Dispatch`, `FnRender`, and browser event
payload types are primarily data shapes; see package docs for their fields.

### App And Page

Use `App` for the normal path. It serves a Neith page, embedded assets, uploads,
and websocket dispatches.

```go
http.HandleFunc("/", neith.App(app, neith.Title("Dashboard")))
```

Use `NewPage` when you want the page as a component or handler.

```go
page := neith.NewPage(
	neith.Title("Admin"),
	neith.Lang("en"),
	neith.Target("main", "app"),
	neith.TargetClass("app-shell"),
	neith.BodyClass("theme-default"),
	neith.Stylesheet("/assets/app.css"),
	neith.Script("/assets/app.js"),
	neith.Head(neith.HTML(`<meta name="theme-color" content="#172026">`)),
	neith.Body(neith.HTML(`<noscript>JavaScript is required.</noscript>`)),
)
http.HandleFunc("/", neith.MiddleWareFn(page.ServeHTTP, app))
```

Render or serve the page directly when useful.

```go
html := neith.RenderComponent(page)
_ = page.Render(ctx, w)
page.ServeHTTP(w, r)
```

`MiddleWareFn` remains available for custom HTTP shells.

```go
http.HandleFunc("/", neith.MiddleWareFn(customShell, app))
```

### View Options

`View` creates an interactive component from any `neith.Component`.

```go
return neith.View(ctx, dashboard(),
	neith.Label("dashboard"),
	neith.OnSubmit(save),
	neith.IntoTag("main"),
)
```

Attach events with classic helpers or the generic `On`.

```go
neith.OnSubmit(save)
neith.OnClick(open)
neith.OnChange(update)
neith.OnInput(search)
neith.OnKeyDown(shortcut)
neith.On(neith.EventPointerDown, dragStart)
```

Short aliases are available when you prefer concise option names.

```go
neith.Submit(save)
neith.Click(open)
neith.Change(update)
neith.Input(search)
neith.KeyDown(shortcut)
```

Choose where rendered HTML lands.

```go
neith.IntoTag("main")
neith.IntoElement("content")
neith.AppendToTag("ul")
neith.PrependToTag("ul")
neith.SwapTagInner("main")
neith.SwapTagOuter("main")
neith.AppendToElement("items")
neith.PrependToElement("items")
neith.SwapElementInner("content")
neith.SwapElementOuter("content")
```

### Components And Dispatch

Raw HTML, templ components, and custom renderers all satisfy `Component`.

```go
type Greeting struct{ Name string }

func (g Greeting) Render(ctx context.Context, w io.Writer) error {
	_, err := fmt.Fprintf(w, "<h1>Hello %s</h1>", html.EscapeString(g.Name))
	return err
}
```

Use `HTML`, `RenderComponent`, `NewFn`, and `View` depending on how close to the
wire you want to be.

```go
raw := neith.HTML(`<p>Ready</p>`)
_, _ = raw.Write([]byte(`<p>More</p>`))
html := neith.RenderComponent(raw, Greeting{Name: "Sean"})
fn := neith.NewFn(ctx, neith.HTML(html)).WithLabel("status")
return neith.View(ctx, fn, neith.IntoTag("main"))
```

The lower-level `FnComponent` methods are still public for explicit dispatch
workflows.

```go
return neith.NewFn(ctx, card()).
	WithContext(ctx).
	WithLabel("card").
	WithEvents(save, neith.EventSubmit).
	SwapTagInner("main")
```

Render-target methods mirror the `View` options.

```go
fn.AppendTag("ul")
fn.PrependTag("ul")
fn.SwapTagInner("main")
fn.SwapTagOuter("main")
fn.AppendElement("items")
fn.PrependElement("items")
fn.SwapElementInner("content")
fn.SwapElementOuter("content")
```

Special dispatches can be returned from handlers.

```go
return neith.FnErr(ctx, err)
return neith.RedirectURL(ctx, "/login")
return neith.NewFn(ctx, nil).WithRedirect("/login")
return neith.NewFn(ctx, nil).WithError(err)
return neith.NewFn(ctx, nil).JS("toast", "Saved")
```

Dispatch an additional effect immediately.

```go
neith.NewFn(ctx, notice()).AppendElement("notifications").Dispatch()
```

`Render` and `Write` exist because `FnComponent` is also a component and writer.

```go
fn := neith.NewFn(ctx, nil)
_, _ = fn.Write([]byte("<p>Buffered</p>"))
_ = fn.Render(ctx, w)
```

### Event Data

Use `EventData` for form maps, form structs, or rich browser event payloads.

```go
values, err := neith.EventData[map[string]string](ctx)
key, err := neith.EventData[neith.KeyboardEvent](ctx)
pointer, err := neith.EventData[neith.PointerEvent](ctx)
mouse, err := neith.EventData[neith.MouseEvent](ctx)
drag, err := neith.EventData[neith.DragEvent](ctx)
touch, err := neith.EventData[neith.TouchEvent](ctx)
_ = []any{values, key, pointer, mouse, drag, touch, err}
```

Use upload and submitter helpers inside submit handlers.

```go
uploads, err := neith.EventUploads(ctx)
submitter, err := neith.EventSubmitter(ctx)
if submitter != nil && submitter.Value == "delete" {
	return neith.RedirectURL(ctx, "/confirm-delete")
}
_ = uploads
```

### DOM Effects

These helpers send immediate browser-side effects from inside a handler.

```go
neith.AddClasses(ctx, "status", "active", "visible")
neith.RemoveClasses(ctx, "status", "pending")
neith.SetAttribute(ctx, "save", "aria-busy", "true")
neith.RemoveAttribute(ctx, "save", "aria-busy")
neith.SetStyle(ctx, "status", "color", "green")
neith.RemoveStyle(ctx, "status", "color")
neith.SetText(ctx, "status", "Saved")
neith.SetValue(ctx, "search", "")
neith.Focus(ctx, "search")
neith.Blur(ctx, "search")
neith.ScrollIntoView(ctx, "results")
neith.Disable(ctx, "save")
neith.Enable(ctx, "save")
neith.RemoveElement(ctx, "modal")
neith.RemoveTag(ctx, "dialog")
neith.JS(ctx, "toast", map[string]string{"message": "Saved"})
```

### Cache

Create and retrieve typed, per-client-session cache values.

```go
cache, err := neith.NewCache(ctx, "count", 0)
cache, err = neith.UseCache[int](ctx, "count")
_ = err
```

Work with cache values and metadata.

```go
_ = cache.Set(cache.Value()+1, time.Minute)
cache.Record(true)
history, ok := cache.History()
created := cache.CreatedAt()
updated := cache.UpdatedAt()
timeout := cache.TimeOut()
expiry := cache.Expiry()
cache.Delete()
_, _, _, _, _, _ = history, ok, created, updated, timeout, expiry
```

Register cache lifecycle callbacks.

```go
neith.OnCacheChange(cache, func() {
	fmt.Println("count changed")
})
neith.OnCacheTimeOut(cache, func() {
	fmt.Println("count expired")
})
```

### Configuration Helpers

Set package configuration early in `main`.

```go
cfg := &neith.Config{
	CacheTimeOut:    10 * time.Minute,
	LogLevel:        neith.Info,
	UploadDir:       "tmp/uploads",
	UploadMaxBytes:  32 << 20,
	UploadMaxMemory: 16 << 20,
}
neith.SetConfig(cfg)
cfg.Set()
```

Use log levels when configuring output.

```go
levels := []neith.LogLevel{
	neith.Debug,
	neith.Info,
	neith.Warn,
	neith.Error,
	neith.Fatal,
	neith.None,
}
_ = levels
```

### UI Components

Import the optional component package:

```go
import "github.com/seanbman/neith/ui"
```

Generic primitives render normal `neith.Component` values.

```go
ui.Element("article", ui.Class("custom"), ui.Text("Hello"))
ui.Fragment(ui.Text("A"), ui.Text("B"))
ui.Text("Escaped <text>")
ui.Panel(ui.Heading("Settings", ui.Level(2)))
ui.Stack(ui.Text("One"), ui.Text("Two"))
ui.Row(ui.Button("Cancel", ui.Secondary()), ui.Button("Save", ui.Primary()))
ui.Grid(ui.Panel("A"), ui.Panel("B"))
ui.Heading("Dashboard")
ui.Alert("Saved")
```

Form controls:

```go
ui.Form(
	ui.HiddenInput("intent", "save"),
	ui.TextInput("title", ui.Label("Title"), ui.Placeholder("Name it")),
	ui.TextArea("message", ui.Label("Message"), ui.Value("Draft")),
	ui.Select("status",
		ui.Label("Status"),
		ui.Options("ok", "queued"),
		ui.Choices(ui.OptionChoice("warning", "Needs attention")),
	),
	ui.Button("Save", ui.Type("submit"), ui.Primary()),
)
```

Tables:

```go
ui.Table(
	ui.Columns("ID", "Status"),
	ui.TableColumns(ui.Column{Header: "Actions"}),
	ui.TableRow("#001", "queued", ui.Button("Open")),
	ui.Rows([]neith.Component{ui.Text("#002"), ui.Text("ok")}),
)
```

### UI Options

Most `ui` components accept options.

```go
ui.Button("Delete",
	ui.ID("delete"),
	ui.Name("intent"),
	ui.Type("submit"),
	ui.Value("delete"),
	ui.Class("button-danger"),
	ui.Attr("data-confirm", "true"),
	ui.BoolAttr("formnovalidate", true),
	ui.Disabled(false),
	ui.Danger(),
)
```

Form and layout options:

```go
ui.Form(
	ui.Method("post"),
	ui.Action("/updates"),
	ui.Required(true),
	ui.Children(ui.Text("Extra child")),
)

ui.TextInput("source",
	ui.Label("Source"),
	ui.LabelClass("field-wide"),
)
```

Write app-specific options with public `ui.Config` methods.

```go
func Dense() ui.Option {
	return func(c *ui.Config) {
		c.Class("is-dense")
		c.Attr("data-density", "dense")
		c.BoolAttr("data-ready", true)
		c.Label("Compact label")
		c.LabelClass("compact-label")
		c.Children(ui.Text("child"))
		c.Choices(ui.Choice{Value: "a", Label: "A"})
		c.Columns(ui.Column{Header: "Name"})
		c.Row("A row")
	}
}
```

## Example App

The repository includes an external consumer-style app in
`examples/readme-setup`. It has its own `go.mod`, imports
`github.com/seanbman/neith`, and uses a local `replace` directive so it runs
against the package source in this repo.

Run it from the repo root:

```sh
make example
```

The example listens on `:8080` by default. If that port is already in use:

```sh
EXAMPLE_ADDR=:8081 make example
```

Run it in debug mode from VS Code with the `Debug README Example` launch
configuration, or start a headless Delve server from the repo root:

```sh
make example-debug
```

Then attach with the `Attach README Example` launch configuration. The default
debug port is `40000`; override it with `DEBUG_PORT=40001 make example-debug`.
The debug target requires Delve:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
```

If `dlv` still is not found after install, add Go's bin directory to your shell:

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Or run it from the example folder:

```sh
cd examples/readme-setup
go run github.com/a-h/templ/cmd/templ@v0.2.513 generate
go run .
```

Set `EXAMPLE_ADDR=:8081` before `go run .` if `:8080` is already in use.

Then open:

```text
http://localhost:8080
```

The example UI keeps its detailed markup in `templ` components in
`examples/readme-setup/dashboard.templ`, builds the add-update form with `ui`
components, then wraps the composed result with `neith.View` in `main.go`. It
also includes `/assets/neith-ui.css` for neutral default styling.
`dashboard_templ.go` is the generated Go file checked in beside it. Regenerate
that file from the repo root with `make example-templ`.

The example serves the package's bundled browser client from `static/assets`, so
it can test local Neith changes without copying generated assets into the
example folder.

## Detailed Notes

Detailed function notes and usage examples live in `notes`:

- `notes/cache/README.md`
- `notes/component/README.md`

## Development

Run Go tests:

```sh
go test ./... -v
```

Run browser-client tests:

```sh
cd static/assets
npm install
npm test
```

Type-check browser-client TypeScript:

```sh
tsc -p static/assets/
```

Build bundled assets:

```sh
make assets
```

## License

MIT
