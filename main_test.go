package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"homer-go/internal/config"
	"homer-go/internal/views"
)

func TestServeHTTPGracefulShutdownDrainsBeforeClosingIdleClientConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	idleClientsClosed := make(chan struct{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(requestStarted)
			<-releaseRequest
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- serveHTTP(ctx, server, listener, time.Second, func() {
			close(idleClientsClosed)
		})
	}()

	requestErr := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			requestErr <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			requestErr <- errors.New(resp.Status)
			return
		}
		requestErr <- nil
	}()

	<-requestStarted
	cancel()

	select {
	case <-idleClientsClosed:
		t.Fatal("idle client connections closed before active request drained")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseRequest)

	select {
	case err := <-requestErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not finish")
	}

	select {
	case <-idleClientsClosed:
	case <-time.After(time.Second):
		t.Fatal("idle client connections were not closed")
	}

	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestPageFromRequestSupportsQueryPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/?page=page2", nil)

	page, ok := pageFromRequest(r)

	if !ok || page != "page2" {
		t.Fatalf("pageFromRequest() = %q, %v; want page2, true", page, ok)
	}
}

func TestEnvUsesProjectScopedVariablesOnly(t *testing.T) {
	t.Setenv("ADDR", "127.0.0.1:1")
	t.Setenv("HOMER_GO_ADDR", "127.0.0.1:2")

	if got := env("HOMER_GO_ADDR", ":8732"); got != "127.0.0.1:2" {
		t.Fatalf("env(HOMER_GO_ADDR) = %q, want project scoped value", got)
	}
	if got := env("HOMER_GO_MISSING", ":8732"); got != ":8732" {
		t.Fatalf("env(HOMER_GO_MISSING) = %q, want fallback", got)
	}
}

func TestServiceWorkerUsesBuildIDAndNoCache(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sw.js", nil)

	assetsDir := t.TempDir()
	writeTestAssets(t, assetsDir)

	serviceWorkerHandler("test-build", assetsDir)(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	body := w.Body.String()
	for _, item := range []string{
		`const CACHE_PREFIX = "homer-go-";`,
		`const CACHE = "homer-go-test-build-`,
		`"/"`,
		`"/assets/manifest.json"`,
		`"/assets/css/webfonts.css"`,
		`"/assets/webfonts/noto/noto-latin-normal.woff2"`,
		`"/assets/icons/homer-go-logo-v2.png"`,
		`"/assets/icons/pwa-192x192.png"`,
		`"/assets/icons/pwa-maskable-512x512.png"`,
		`"/assets/icons/apple-touch-icon.png"`,
		`event.request.mode === "navigate"`,
		`const precached = ASSETS.includes(url.pathname);`,
		`if (!precached)`,
		`appPath.startsWith("/fragments/")`,
		`key.startsWith(CACHE_PREFIX) && key !== CACHE`,
		`function cacheResponse(request, response)`,
		`if (!response.ok || response.type !== "basic") return Promise.resolve();`,
		`const responsePromise = fetch(event.request);`,
		`event.waitUntil(`,
		`.then(response => cacheResponse(event.request, response))`,
		`return responsePromise;`,
		`function fetchNavigation(event)`,
		`const responsePromise = fetch(event.request, { cache: "reload" });`,
		`fetchNavigation(event)`,
		`cached || caches.match(BASE_PATH + "/")`,
		`cached || fetchAndCache(event)`,
		`self.skipWaiting()`,
		`self.clients.claim()`,
	} {
		if !strings.Contains(body, item) {
			t.Fatalf("service worker missing %s: %s", item, body)
		}
	}
}

func TestServiceWorkerAppliesBasePath(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dash/sw.js", nil)

	assetsDir := t.TempDir()
	writeTestAssets(t, assetsDir)

	serviceWorkerHandlerWithPaths("test-build", assetsDir, views.NewPaths("/dash"))(w, r)

	body := w.Body.String()
	for _, item := range []string{
		`const BASE_PATH = "/dash";`,
		`"/dash/assets/manifest.json"`,
		`"/dash/assets/homer-go.js"`,
		`url.pathname.startsWith(BASE_PATH + "/")`,
		`const appPath = BASE_PATH ? url.pathname.slice(BASE_PATH.length) || "/" : url.pathname;`,
		`caches.match(BASE_PATH + "/")`,
	} {
		if !strings.Contains(body, item) {
			t.Fatalf("service worker missing %s: %s", item, body)
		}
	}
}

func TestServiceWorkerRejectsNonGetAndHead(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/sw.js", nil)

	serviceWorkerHandler("test-build", t.TempDir())(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestCacheNameChangesWhenAssetsChange(t *testing.T) {
	assetsDir := t.TempDir()
	writeTestAssets(t, assetsDir)

	assets := []string{"/", "/assets/homer-go.js"}
	first := cacheName("dev", assetsDir, assets)

	if err := os.WriteFile(filepath.Join(assetsDir, "homer-go.js"), []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	second := cacheName("dev", assetsDir, assets)

	if first == second {
		t.Fatalf("cacheName did not change after asset content changed: %q", first)
	}
}

func writeTestAssets(t *testing.T, assetsDir string) {
	t.Helper()
	files := map[string]string{
		"manifest.json":                                        "{}",
		"homer-go.css":                                         "",
		"homer-go.js":                                          "old",
		"vendor/bulma/css/bulma.min.css":                       "",
		"css/themes/classic.css":                               "",
		"css/webfonts.css":                                     "",
		"css/base.css":                                         "",
		"css/status.css":                                       "",
		"css/highlights.css":                                   "",
		"css/themes/neon.css":                                  "",
		"css/themes/walkxcode.css":                             "",
		"webfonts/noto/noto-latin-normal.woff2":                "",
		"vendor/fontawesome/css/all.min.css":                   "",
		"vendor/fontawesome/webfonts/fa-brands-400.woff2":      "",
		"vendor/fontawesome/webfonts/fa-regular-400.woff2":     "",
		"vendor/fontawesome/webfonts/fa-solid-900.woff2":       "",
		"vendor/fontawesome/webfonts/fa-v4compatibility.woff2": "",
		"icons/homer-go-logo-v2.png":                           "",
		"icons/pwa-192x192.png":                                "",
		"icons/pwa-512x512.png":                                "",
		"icons/pwa-maskable-512x512.png":                       "",
		"icons/apple-touch-icon.png":                           "",
		"icons/favicon-32x32.png":                              "",
		"icons/favicon-16x16.png":                              "",
	}
	for name, body := range files {
		path := filepath.Join(assetsDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
}

func TestPageFromRequestSupportsPagePath(t *testing.T) {
	r := httptest.NewRequest("GET", "/page/additional-page", nil)

	page, ok := pageFromRequest(r)

	if !ok || page != "additional-page" {
		t.Fatalf("pageFromRequest() = %q, %v; want additional-page, true", page, ok)
	}
}

func TestPageFromRequestRejectsUnsafePage(t *testing.T) {
	tests := []string{
		"/?page=../secret",
		"/?page=a/b",
		"/page/..%2Fsecret",
	}

	for _, target := range tests {
		r := httptest.NewRequest("GET", target, nil)
		if page, ok := pageFromRequest(r); ok {
			t.Fatalf("pageFromRequest(%q) = %q, true; want rejected", target, page)
		}
	}
}

func TestPageFromRequestDefaultsToDefault(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	page, ok := pageFromRequest(r)

	if !ok || page != "default" {
		t.Fatalf("pageFromRequest() = %q, %v; want default, true", page, ok)
	}
}

func TestRenderConfigLoadErrorShowsGetStartedForMissingConfig(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	err := &config.LoadError{Kind: config.ErrorNotFound, Source: "assets/config.yml", Err: os.ErrNotExist}

	renderConfigLoadError(w, r, err)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if body := w.Body.String(); !strings.Contains(body, "No configuration found!") {
		t.Fatalf("body = %q, want get started message", body)
	}
}

func TestRenderConfigLoadErrorShowsParseErrorPage(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	err := &config.LoadError{Kind: config.ErrorParse, Source: "assets/config.yml", Err: errors.New("yaml: line 1: bad")}

	renderConfigLoadError(w, r, err)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Error parsing configuration") || !strings.Contains(body, "yaml: line 1: bad") {
		t.Fatalf("body = %q, want parse error page", body)
	}
}

func TestRenderConfigLoadErrorReturnsNotFoundForMissingPageConfig(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/page/team", nil)
	err := &config.LoadError{Kind: config.ErrorPageNotFound, Source: "assets/team.yml", Err: os.ErrNotExist}

	renderConfigLoadError(w, r, err)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if body := w.Body.String(); strings.Contains(body, "No configuration found!") {
		t.Fatalf("body = %q, did not want get started message", body)
	}
}

func TestRenderConfigLoadErrorShowsExternalErrorPage(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	err := &config.LoadError{Kind: config.ErrorExternal, Source: "https://example.test/config.yml", Err: errors.New("502 Bad Gateway")}

	renderConfigLoadError(w, r, err)

	if w.Code != 502 {
		t.Fatalf("status = %d, want 502", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Error loading external configuration") || !strings.Contains(body, "502 Bad Gateway") {
		t.Fatalf("body = %q, want external error page", body)
	}
}
