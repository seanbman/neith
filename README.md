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

## Features

- Render Go components into the DOM over WebSockets.
- Attach server-side handlers to browser events.
- Swap, append, prepend, or remove elements by tag or ID.
- Add and remove CSS classes from the server.
- Redirect the browser from a handler.
- Run custom JavaScript functions from Go.
- Read typed event payloads with `EventData[T]`.
- Inspect uploaded files and form submitter metadata from handlers.
- Store per-connection state with generic caches.
- Configure logging and cache expiry.

## Installation

```sh
go get github.com/seanbman/neith
```

Neith requires Go 1.21 or newer.

## Quick Start

Serve an HTML shell with a `<main>` element and the bundled browser client, then wrap
your route with `neith.MiddleWareFn`.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/seanbman/neith"
)

func page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html>
	<head>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<script defer src="/assets/index.min.js"></script>
	</head>
	<body>
		<main></main>
	</body>
</html>`)
}

func app(ctx context.Context) neith.FnComponent {
	return neith.NewFn(ctx, neith.HTML(`
		<button>Click me</button>
	`)).WithEvents(clicked, neith.OnClick)
}

func clicked(ctx context.Context) neith.FnComponent {
	return neith.NewFn(ctx, neith.HTML(`
		<section>
			<h1>Hello from the server</h1>
			<p>The button click was handled in Go.</p>
		</section>
	`)).SwapTagInner("main")
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("static/assets"))))
	http.HandleFunc("/", neith.MiddleWareFn(page, app))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Open `http://localhost:8080`. The first request serves the HTML shell. The browser
client then opens a WebSocket back to the same route with `?neith_id=...`, and
`MiddleWareFn` sends the initial `FnComponent` to the client.

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

## Events

Attach one or more DOM events with `WithEvents`.

```go
func form(ctx context.Context) neith.FnComponent {
	return neith.NewFn(ctx, neith.HTML(`
		<form>
			<input name="name" placeholder="Name">
			<button>Save</button>
		</form>
	`)).WithEvents(save, neith.OnSubmit)
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

## Cache

Neith includes a generic, per-connection cache. Create a cache once, then reuse it
from later handlers for the same client connection.

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
	}, neith.OnClick)
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

## Example App

The repository includes an external consumer-style app in
`examples/readme-setup`. It has its own `go.mod`, imports
`github.com/seanbman/neith`, and uses a local `replace` directive so it runs
against the package source in this repo.

Run it from the repo root:

```sh
make example
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

Then open:

```text
http://localhost:8080
```

The example UI is written as `templ` components in
`examples/readme-setup/dashboard.templ`; `dashboard_templ.go` is the generated Go
file checked in beside it. Regenerate that file from the repo root with
`make example-templ`.

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
