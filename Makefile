.PHONY: templ example example-debug

VERSION = 0.4
ESBUILD = ./es-build
GOPATH_BIN = $(shell go env GOPATH)/bin
DLV ?= $(GOPATH_BIN)/dlv
DEBUG_PORT ?= 40000

make:
	go run .

test:
	go test ./... -v
	cd static/assets && npm install && npm test

example:
	cd examples/readme-setup && go run .

example-debug:
	@command -v $(DLV) >/dev/null || (echo "Delve is required. Install it with: go install github.com/go-delve/delve/cmd/dlv@latest" && exit 1)
	cd examples/readme-setup && $(DLV) debug . --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient

coverage:
	go test ./... -v -coverprofile=cover.out
	go tool cover -html=coverage.out

assets: templ
	tsc -p "static/assets/"
	$(ESBUILD) static/assets/index.ts --bundle --minify --outfile=static/assets/index.min.js
	./tailwindcss -i static/assets/stylesheets/tailwind.css -o static/assets/stylesheets/tailwind.min.css --minify
	sass static/assets/sass:static/assets/stylesheets

bundle:
	$(ESBUILD) static/assets/index.ts --bundle --minify --outfile=static/assets/index.min.js

templ:
	/Users/seanburman/go/bin/templ generate

tsc:
	tsc -p "static/assets/" --watch

tailwind:
	./tailwindcss -i static/assets/stylesheets/tailwind.css -o static/assets/stylesheets/tailwind.min.css --watch --minify

sass:
	sass --watch static/assets/sass:static/assets/stylesheets

publish:
	git tag -s v0.3.305 -m "fcmp v$(VERSION)" && \
	git push --tags && \
	GOPROXY=proxy.golang.org go list -m github.com/seanbman/fcmp@v$(VERSION) && \
	curl https://sum.golang.org/lookup/github.com/seanbman/fcmp@v$(VERSION)
