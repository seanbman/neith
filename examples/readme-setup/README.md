# Neith README Setup

This folder mimics a small app created by a programmer using Neith from the
outside.

It has its own Go module and imports:

```go
github.com/seanbman/neith
```

For local package development, `go.mod` uses:

```go
replace github.com/seanbman/neith => ../..
```

Run it from this folder:

```sh
go mod tidy
go run github.com/a-h/templ/cmd/templ@v0.2.513 generate
go run .
```

Run it in debug mode from the repo root with VS Code's `Debug README Example`
launch configuration, or start a headless Delve server:

```sh
make example-debug
```

Then attach with VS Code's `Attach README Example` launch configuration. The
default debug port is `40000`; override it with
`DEBUG_PORT=40001 make example-debug`. The debug target requires Delve:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
```

If `dlv` still is not found after install, add Go's bin directory to your shell:

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Then open:

```text
http://localhost:8080
```

The example renders a small admin-style cache monitor with `templ` components in
`dashboard.templ`. Fill out the form and submit it to add a row to the table.
Use a row's `Delete` button to remove that record from the current cache value.
Each add or delete stores a new value in the client-session Neith cache, records
a cache history snapshot, and re-renders the table.

Beneath the table, one terminal-style panel shows the full literal contents of
the current `admin_updates` cache. A second terminal-style panel shows the
history store. History is separate from the current cache value: the current
cache is the latest `admin_updates` slice, while history is a timestamp-keyed
set of older recorded versions created when `Record(true)` is enabled and
`Set(...)` runs.

The app follows the README quick-start shape:

```text
examples/readme-setup/
├── dashboard.templ
├── dashboard_templ.go
├── go.mod
├── main.go
└── static/
    └── index.html
```

`dashboard_templ.go` is generated from `dashboard.templ`. Regenerate it from the
repo root with `make example-templ`, or from this folder with:

```sh
go run github.com/a-h/templ/cmd/templ@v0.2.513 generate
```

`main.go` serves the package's bundled browser client from `../../static/assets`
so the example can test local Neith changes without copying generated assets.
