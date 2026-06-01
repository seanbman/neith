# fcmp

`fcmp` is a small Go package for building server-rendered, interactive HTML components.
It wraps Go components with a WebSocket-backed dispatch layer so the server can render
HTML, respond to DOM events, update elements, redirect the browser, call client-side
JavaScript, and keep per-client state.

The package is designed around a minimal component interface:

```go
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}
```

That means plain `fcmp.HTML`, `templ` components, or your own renderable types can be
used as fcmp components.

## Features

- Render Go components into the DOM over WebSockets.
- Attach server-side handlers to browser events.
- Swap, append, prepend, or remove elements by tag or ID.
- Add and remove CSS classes from the server.
- Redirect the browser from a handler.
- Run custom JavaScript functions from Go.
- Read typed event payloads with `EventData[T]`.
- Store per-connection state with generic caches.
- Configure logging and cache expiry.

## Installation

```sh
go get github.com/snburman/fcmp
```

`fcmp` requires Go 1.21 or newer.

## Quick Start

Serve an HTML shell with a `<main>` element and the bundled browser client, then wrap
your route with `fcmp.MiddleWareFn`.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/snburman/fcmp"
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

func app(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`
		<button>Click me</button>
	`)).WithEvents(clicked, fcmp.OnClick)
}

func clicked(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`
		<section>
			<h1>Hello from the server</h1>
			<p>The button click was handled in Go.</p>
		</section>
	`)).SwapTagInner("main")
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("static/assets"))))
	http.HandleFunc("/", fcmp.MiddleWareFn(page, app))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Open `http://localhost:8080`. The first request serves the HTML shell. The browser
client then opens a WebSocket back to the same route with `?fcmp_id=...`, and
`MiddleWareFn` sends the initial `FnComponent` to the client.

## Rendering Components

Create an `FnComponent` with `NewFn(ctx, component)`. By default, it swaps the inner
HTML of the first `<main>` tag.

```go
func view(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`<h1>Dashboard</h1>`))
}
```

You can change where the rendered HTML is applied:

```go
fcmp.NewFn(ctx, component).SwapTagInner("main")
fcmp.NewFn(ctx, component).SwapTagOuter("main")
fcmp.NewFn(ctx, component).AppendTag("ul")
fcmp.NewFn(ctx, component).PrependTag("ul")

fcmp.NewFn(ctx, component).SwapElementInner("content")
fcmp.NewFn(ctx, component).SwapElementOuter("content")
fcmp.NewFn(ctx, component).AppendElement("items")
fcmp.NewFn(ctx, component).PrependElement("items")
```

To render components to a string outside the live dispatch flow:

```go
html := fcmp.RenderComponent(fcmp.HTML(`<p>Rendered</p>`))
```

## Events

Attach one or more DOM events with `WithEvents`.

```go
func form(ctx context.Context) fcmp.FnComponent {
	return fcmp.NewFn(ctx, fcmp.HTML(`
		<form>
			<input name="name" placeholder="Name">
			<button>Save</button>
		</form>
	`)).WithEvents(save, fcmp.OnSubmit)
}

func save(ctx context.Context) fcmp.FnComponent {
	values, err := fcmp.EventData[map[string]string](ctx)
	if err != nil {
		return fcmp.FnErr(ctx, err)
	}

	return fcmp.NewFn(ctx, fcmp.HTML(
		"<p>Saved " + values["name"] + "</p>",
	)).SwapTagInner("main")
}
```

For pointer, mouse, keyboard, drag, touch, input, change, submit, and other DOM events,
the browser client sends event data back to Go. Use `EventData[T]` with the matching
type, such as `fcmp.PointerEvent`, `fcmp.DragEvent`, or your own form-data struct/map.

## Server-Initiated Updates

Handlers can dispatch extra effects while handling an event:

```go
func clicked(ctx context.Context) fcmp.FnComponent {
	fcmp.AddClasses(ctx, "status", "active")
	fcmp.RemoveClasses(ctx, "status", "pending")
	fcmp.JS(ctx, "Testing", "called from Go")

	return fcmp.NewFn(ctx, fcmp.HTML(`<p id="status">Done</p>`))
}
```

Other helpers:

```go
fcmp.RemoveElement(ctx, "modal")
fcmp.RemoveTag(ctx, "dialog")
return fcmp.RedirectURL(ctx, "/next")
```

## Cache

`fcmp` includes a generic, per-connection cache. Create a cache once, then reuse it
from later handlers for the same client connection.

```go
func app(ctx context.Context) fcmp.FnComponent {
	_, _ = fcmp.NewCache(ctx, "count", 0)
	return counter(ctx)
}

func counter(ctx context.Context) fcmp.FnComponent {
	count, err := fcmp.UseCache[int](ctx, "count")
	if err != nil {
		return fcmp.FnErr(ctx, err)
	}

	_ = count.Set(count.Value() + 1)

	return fcmp.NewFn(ctx, fcmp.HTML(fmt.Sprintf(
		`<button>Clicked %d times</button>`,
		count.Value(),
	))).WithEvents(func(ctx context.Context) fcmp.FnComponent {
		return counter(ctx)
	}, fcmp.OnClick)
}
```

Cache helpers include:

- `Set(value, timeout...)`
- `Value()`
- `Delete()`
- `CreatedAt()`
- `UpdatedAt()`
- `Expiry()`
- `OnCacheChange(cache, fn)`
- `OnCacheTimeOut(cache, fn)`

## Configuration

The default config uses a 30 minute cache timeout and logs errors.

```go
fcmp.SetConfig(&fcmp.Config{
	CacheTimeOut: 10 * time.Minute,
	LogLevel:     fcmp.Info,
})
```

Set `Silent: true` or `LogLevel: fcmp.None` to disable package logs.

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

Build bundled assets:

```sh
make assets
```

## License

MIT
