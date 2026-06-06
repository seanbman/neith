package neith

import (
	"bytes"
	"context"
	"embed"
	"html"
	"io"
	"net/http"
	"time"
)

//go:embed static/assets/neith.min.js static/assets/neith-ui.css
var embeddedAssets embed.FS

// Page is the default HTML document Neith serves for an app.
//
// It includes a render target, the bundled browser client, and the optional
// neutral UI stylesheet. Customize it with PageOption values when mounting an
// app with App or when passing the page to MiddleWareFn directly.
type Page struct {
	Title       string
	TargetTag   string
	TargetID    string
	Lang        string
	Styles      []string
	Scripts     []string
	Head        []Component
	Body        []Component
	BodyClass   string
	TargetClass string
}

// PageOption customizes the default Neith page.
type PageOption func(*Page)

// NewPage creates a default Neith page.
func NewPage(opts ...PageOption) Page {
	p := Page{
		Title:     "Neith",
		TargetTag: "main",
		Lang:      "en",
		Styles:    []string{"/assets/neith-ui.css"},
		Scripts:   []string{"/assets/neith.min.js"},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}
	return p
}

// App mounts a Neith app with the default page and embedded assets.
func App(hf HandleFn, opts ...PageOption) http.HandlerFunc {
	page := NewPage(opts...)
	app := MiddleWareFn(page.ServeHTTP, hf)
	return func(w http.ResponseWriter, r *http.Request) {
		if serveEmbeddedAsset(w, r) {
			return
		}
		app(w, r)
	}
}

// ServeHTTP writes the page as an HTTP response.
func (p Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = p.Render(r.Context(), w)
}

// Render writes the page HTML.
func (p Page) Render(ctx context.Context, w io.Writer) error {
	if p.TargetTag == "" {
		p.TargetTag = "main"
	}
	if p.Lang == "" {
		p.Lang = "en"
	}
	if _, err := io.WriteString(w, "<!doctype html><html lang=\""+html.EscapeString(p.Lang)+"\"><head>"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, `<meta name="viewport" content="width=device-width, initial-scale=1">`); err != nil {
		return err
	}
	if p.Title != "" {
		if _, err := io.WriteString(w, "<title>"+html.EscapeString(p.Title)+"</title>"); err != nil {
			return err
		}
	}
	for _, href := range p.Styles {
		if href == "" {
			continue
		}
		if _, err := io.WriteString(w, `<link rel="stylesheet" href="`+html.EscapeString(href)+`">`); err != nil {
			return err
		}
	}
	for _, child := range p.Head {
		if err := child.Render(ctx, w); err != nil {
			return err
		}
	}
	for _, src := range p.Scripts {
		if src == "" {
			continue
		}
		if _, err := io.WriteString(w, `<script defer src="`+html.EscapeString(src)+`"></script>`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "</head><body"+classAttr(p.BodyClass)+">"); err != nil {
		return err
	}
	for _, child := range p.Body {
		if err := child.Render(ctx, w); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "<"+p.TargetTag+p.targetAttrs()+"></"+p.TargetTag+">"); err != nil {
		return err
	}
	_, err := io.WriteString(w, "</body></html>")
	return err
}

// Title sets the page title.
func Title(title string) PageOption {
	return func(p *Page) {
		p.Title = title
	}
}

// Lang sets the page language attribute.
func Lang(lang string) PageOption {
	return func(p *Page) {
		p.Lang = lang
	}
}

// Target sets the element tag and optional ID Neith renders into.
func Target(tag string, id string) PageOption {
	return func(p *Page) {
		p.TargetTag = tag
		p.TargetID = id
	}
}

// Stylesheet appends a stylesheet URL.
func Stylesheet(href string) PageOption {
	return func(p *Page) {
		p.Styles = append(p.Styles, href)
	}
}

// Script appends a deferred script URL.
func Script(src string) PageOption {
	return func(p *Page) {
		p.Scripts = append(p.Scripts, src)
	}
}

// ClientScript sets the Neith browser client script URL.
func ClientScript(src string) PageOption {
	return func(p *Page) {
		p.Scripts = []string{src}
	}
}

// Head appends components into the document head.
func Head(children ...Component) PageOption {
	return func(p *Page) {
		p.Head = append(p.Head, children...)
	}
}

// Style appends an inline stylesheet to the document head.
func Style(css string) PageOption {
	return Head(HTML("<style>" + css + "</style>"))
}

// Body appends components before the Neith render target.
func Body(children ...Component) PageOption {
	return func(p *Page) {
		p.Body = append(p.Body, children...)
	}
}

// BodyClass sets the page body class attribute.
func BodyClass(class string) PageOption {
	return func(p *Page) {
		p.BodyClass = class
	}
}

// TargetClass sets the render target class attribute.
func TargetClass(class string) PageOption {
	return func(p *Page) {
		p.TargetClass = class
	}
}

func (p Page) targetAttrs() string {
	attrs := classAttr(p.TargetClass)
	if p.TargetID != "" {
		attrs += ` id="` + html.EscapeString(p.TargetID) + `"`
	}
	return attrs
}

func classAttr(class string) string {
	if class == "" {
		return ""
	}
	return ` class="` + html.EscapeString(class) + `"`
}

func serveEmbeddedAsset(w http.ResponseWriter, r *http.Request) bool {
	var path string
	var contentType string

	switch r.URL.Path {
	case "/assets/neith.min.js":
		path = "static/assets/neith.min.js"
		contentType = "text/javascript; charset=utf-8"
	case "/assets/neith-ui.css":
		path = "static/assets/neith-ui.css"
		contentType = "text/css; charset=utf-8"
	default:
		return false
	}

	body, err := embeddedAssets.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, r.URL.Path, time.Time{}, bytes.NewReader(body))
	return true
}
