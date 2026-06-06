package neith

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageRenderDefaults(t *testing.T) {
	html := RenderComponent(NewPage(Title("Demo")))

	for _, want := range []string{
		`<!doctype html>`,
		`<title>Demo</title>`,
		`<link rel="stylesheet" href="/assets/neith-ui.css">`,
		`<script defer src="/assets/neith.min.js"></script>`,
		`<main></main>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected page to contain %q, got %s", want, html)
		}
	}
}

func TestPageRenderCustomTargetAndHead(t *testing.T) {
	html := RenderComponent(NewPage(
		Target("section", "app"),
		TargetClass("shell"),
		ClientScript("/static/assets/neith.min.js"),
		Head(HTML(`<meta name="theme-color" content="#172026">`)),
		Style(`body { color: #172026; }`),
	))

	for _, want := range []string{
		`<meta name="theme-color" content="#172026">`,
		`<style>body { color: #172026; }</style>`,
		`<script defer src="/static/assets/neith.min.js"></script>`,
		`<section class="shell" id="app"></section>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected page to contain %q, got %s", want, html)
		}
	}
	if strings.Contains(html, `<script defer src="/assets/neith.min.js"></script>`) {
		t.Fatalf("expected ClientScript to replace the default script, got %s", html)
	}
}

func TestAppServesEmbeddedAssets(t *testing.T) {
	handler := App(func(ctx context.Context) FnComponent {
		return View(ctx, HTML("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/neith-ui.css", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "--n-ui-primary-bg") {
		t.Fatalf("expected embedded stylesheet, got %s", rec.Body.String())
	}
}

func TestEmbeddedAssetsOnlyClaimNeithFiles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)
	rec := httptest.NewRecorder()

	if serveEmbeddedAsset(rec, req) {
		t.Fatal("expected app-owned asset path to be left for the app")
	}
}
