# fcmp README Setup

This folder mimics a small app created by a programmer using `fcmp` from the
outside.

It has its own Go module and imports:

```go
github.com/snburman/fcmp
```

For local package development, `go.mod` uses:

```go
replace github.com/snburman/fcmp => ../..
```

Run it from this folder:

```sh
go mod tidy
go run .
```

Then open:

```text
http://localhost:8080
```

The app follows the README quick-start shape:

```text
examples/readme-setup/
├── go.mod
├── main.go
└── static/
    └── index.html
```

`main.go` serves the package's bundled browser client from `../../static/assets`
so the example can test local `fcmp` changes without copying generated assets.
