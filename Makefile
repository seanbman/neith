.PHONY: templ example-templ example example-debug

VERSION ?= 0.4.0
TAG ?= v$(VERSION)
TAG_FLAGS ?= -s
ESBUILD = ./es-build
GOPATH_BIN = $(shell go env GOPATH)/bin
DLV ?= $(GOPATH_BIN)/dlv
DEBUG_PORT ?= 40000
TEMPL_VERSION ?= v0.2.513

make:
	go run .

test:
	go test ./... -v
	cd static/assets && npm install && npm test

example:
	cd examples/readme-setup && go run .

example-templ:
	cd examples/readme-setup && go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate

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
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate

tsc:
	tsc -p "static/assets/" --watch

tailwind:
	./tailwindcss -i static/assets/stylesheets/tailwind.css -o static/assets/stylesheets/tailwind.min.css --watch --minify

sass:
	sass --watch static/assets/sass:static/assets/stylesheets

publish:
	git tag $(TAG_FLAGS) $(TAG) -m "neith $(TAG)" && \
	git push origin $(TAG) && \
	GOPROXY=proxy.golang.org go list -m github.com/seanbman/neith@$(TAG) && \
	curl https://sum.golang.org/lookup/github.com/seanbman/neith@$(TAG)
