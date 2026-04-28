package config

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeColumnsAllowsOnlyHomerValues(t *testing.T) {
	tests := map[any]string{
		nil:    "3",
		"auto": "auto",
		1:      "1",
		"2":    "2",
		3:      "3",
		"4":    "4",
		6:      "6",
		"12":   "12",
		"5":    "3",
		5:      "3",
		"bad":  "3",
	}

	for input, want := range tests {
		cfg, err := decode(deepMerge(defaultMap(), map[string]any{"columns": input}))
		if err != nil {
			t.Fatalf("decode() error = %v", err)
		}
		if cfg.Columns != want {
			t.Fatalf("columns %v decoded as %q, want %q", input, cfg.Columns, want)
		}
	}
}

func TestUnsupportedConfigPathsReportsConfiguredUnsupportedKeys(t *testing.T) {
	raw := deepMerge(defaultMap(), map[string]any{
		"hotkey": map[string]any{"search": "Shift"},
		"proxy": map[string]any{
			"useCredentials": true,
			"headers":        map[string]any{"Authorization": "Bearer test"},
		},
		"services": []any{
			map[string]any{
				"name": "Group",
				"items": []any{
					map[string]any{
						"name":             "App",
						"type":             "Ping",
						"url":              "https://example.test",
						"useCredentials":   false,
						"headers":          map[string]any{"X-Test": "1"},
						"timeout":          500,
						"updateIntervalMs": 30000,
						"checkInterval":    10000,
					},
				},
			},
		},
	})
	cfg, err := decode(raw)
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}

	got := UnsupportedConfigPaths(cfg)
	want := []string{
		"hotkey",
		"proxy.useCredentials",
		"services[0].items[0].checkInterval",
		"services[0].items[0].timeout",
		"services[0].items[0].updateIntervalMs",
		"services[0].items[0].useCredentials",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("UnsupportedConfigPaths() = %#v, want %#v", got, want)
	}
}

func TestUnsupportedConfigPathsIgnoresDefaults(t *testing.T) {
	cfg, err := decode(defaultMap())
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}

	if got := UnsupportedConfigPaths(cfg); len(got) != 0 {
		t.Fatalf("UnsupportedConfigPaths() = %#v, want none", got)
	}
}

func TestDecodeProxyAndItemHeaders(t *testing.T) {
	raw := deepMerge(defaultMap(), map[string]any{
		"proxy": map[string]any{
			"headers": map[string]any{
				"Authorization": "Bearer global",
				"X-Number":      42,
			},
		},
		"services": []any{
			map[string]any{
				"items": []any{
					map[string]any{
						"name": "App",
						"url":  "https://example.test",
						"headers": map[string]any{
							"Authorization": "Bearer item",
						},
					},
				},
			},
		},
	})
	cfg, err := decode(raw)
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}

	if cfg.Proxy.Headers["Authorization"] != "Bearer global" || cfg.Proxy.Headers["X-Number"] != "42" {
		t.Fatalf("proxy headers = %#v, want decoded string headers", cfg.Proxy.Headers)
	}
	item := cfg.Services[0].Items[0]
	if item.Headers["Authorization"] != "Bearer item" {
		t.Fatalf("item headers = %#v, want decoded item headers", item.Headers)
	}
}

func TestReadYAMLSupportsMergeKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	body := []byte(`
defaults: &item_defaults
  target: _blank
  icon: fas fa-server
  subtitle: Inherited

services:
  - name: Group
    items:
      - <<: *item_defaults
        name: App
        url: https://example.test
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	raw, err := readYAML(context.Background(), path, false)
	if err != nil {
		t.Fatalf("readYAML() error = %v", err)
	}
	cfg, err := decode(deepMerge(defaultMap(), raw))
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}
	if got := cfg.Services[0].Items[0].Target; got != "_blank" {
		t.Fatalf("merged target = %q, want _blank", got)
	}
	if got := cfg.Services[0].Items[0].Icon; got != "fas fa-server" {
		t.Fatalf("merged icon = %q, want fas fa-server", got)
	}
	if got := cfg.Services[0].Items[0].Subtitle; got != "Inherited" {
		t.Fatalf("merged subtitle = %q, want Inherited", got)
	}
}

func TestLoadReturnsNotFoundErrorForMissingConfig(t *testing.T) {
	_, err := (Loader{AssetsDir: t.TempDir()}).Load(context.Background(), "default")
	if err == nil {
		t.Fatal("Load() error = nil, want not found error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load() error = %T, want LoadError", err)
	}
	if loadErr.Kind != ErrorNotFound {
		t.Fatalf("LoadError.Kind = %q, want %q", loadErr.Kind, ErrorNotFound)
	}
}

func TestLoadAutoInitializesMissingWorkdirConfigFromExample(t *testing.T) {
	configDir := t.TempDir()
	var initialized string
	cfg, err := (Loader{
		ConfigDir:     configDir,
		ExampleConfig: []byte("title: Generated\nservices: []\n"),
		AutoInit:      true,
		OnInit: func(path string) {
			initialized = path
		},
	}).Load(context.Background(), "default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	path := filepath.Join(configDir, "config.yml")
	if initialized != path {
		t.Fatalf("OnInit path = %q, want %q", initialized, path)
	}
	if cfg.Title != "Generated" {
		t.Fatalf("title = %q, want Generated", cfg.Title)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "title: Generated\nservices: []\n" {
		t.Fatalf("generated config = %q, want embedded example", body)
	}
}

func TestLoadAutoInitReturnsExplicitErrorForMissingConfigDir(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "missing")
	_, err := (Loader{
		ConfigDir:     configDir,
		ExampleConfig: []byte("title: Generated\nservices: []\n"),
		AutoInit:      true,
	}).Load(context.Background(), "default")
	if err == nil {
		t.Fatal("Load() error = nil, want config directory error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load() error = %T, want LoadError", err)
	}
	if loadErr.Kind != ErrorConfigDir {
		t.Fatalf("LoadError.Kind = %q, want %q", loadErr.Kind, ErrorConfigDir)
	}
	if !strings.Contains(loadErr.Error(), "create the directory first") {
		t.Fatalf("LoadError.Error() = %q, want explicit directory guidance", loadErr.Error())
	}
	if _, statErr := os.Stat(configDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("config directory stat error = %v, want still missing", statErr)
	}
}

func TestLoadDoesNotOverwriteInvalidConfig(t *testing.T) {
	configDir := t.TempDir()
	path := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(path, []byte("title: [broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := (Loader{
		ConfigDir:     configDir,
		ExampleConfig: []byte("title: Generated\n"),
		AutoInit:      true,
	}).Load(context.Background(), "default")
	if err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}

	body, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(body) != "title: [broken" {
		t.Fatalf("config was overwritten: %q", body)
	}
	var loadErr *LoadError
	if !errors.As(err, &loadErr) || loadErr.Kind != ErrorParse {
		t.Fatalf("Load() error = %v, want parse LoadError", err)
	}
}

func TestLoadReturnsPageNotFoundErrorForMissingPageConfig(t *testing.T) {
	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), []byte("title: Dashboard\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "team")
	if err == nil {
		t.Fatal("Load() error = nil, want page not found error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load() error = %T, want LoadError", err)
	}
	if loadErr.Kind != ErrorPageNotFound {
		t.Fatalf("LoadError.Kind = %q, want %q", loadErr.Kind, ErrorPageNotFound)
	}
}

func TestLoadReturnsParseErrorForInvalidYAML(t *testing.T) {
	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), []byte("title: [broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load() error = %T, want LoadError", err)
	}
	if loadErr.Kind != ErrorParse {
		t.Fatalf("LoadError.Kind = %q, want %q", loadErr.Kind, ErrorParse)
	}
	if !strings.Contains(loadErr.Error(), "could not be parsed") {
		t.Fatalf("LoadError.Error() = %q, want parse message", loadErr.Error())
	}
}

func TestLoadReturnsExternalErrorWhenExternalConfigFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	assetsDir := t.TempDir()
	body := []byte("externalConfig: " + server.URL + "\n")
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err == nil {
		t.Fatal("Load() error = nil, want external error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load() error = %T, want LoadError", err)
	}
	if loadErr.Kind != ErrorExternal {
		t.Fatalf("LoadError.Kind = %q, want %q", loadErr.Kind, ErrorExternal)
	}
	if !strings.Contains(loadErr.Error(), "external configuration") {
		t.Fatalf("LoadError.Error() = %q, want external message", loadErr.Error())
	}
}

func TestLoadDoesNotResolveRemoteMessage(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"title":"Remote title"}`))
	}))
	defer server.Close()

	assetsDir := t.TempDir()
	body := []byte(`
message:
  url: ` + server.URL + `
  title: Local title
`)
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if called {
		t.Fatal("Load() requested remote message, want message resolution to be explicit")
	}
	if cfg.Message.Title != "Local title" {
		t.Fatalf("message title = %q, want Local title", cfg.Message.Title)
	}
}

func TestResolveMessageUsesRemoteMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept header = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "Remote title",
			"style": null,
			"content": "",
			"icon": "fas fa-bell"
		}`))
	}))
	defer server.Close()

	assetsDir := t.TempDir()
	body := []byte(`
message:
  url: ` + server.URL + `
  style: is-warning
  title: Local title
  icon: fas fa-info
  content: Local content
`)
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Message = ResolveMessage(context.Background(), cfg.Message)

	if cfg.Message.Title != "Remote title" {
		t.Fatalf("message title = %q, want Remote title", cfg.Message.Title)
	}
	if cfg.Message.Style != "is-warning" {
		t.Fatalf("message style = %q, want local fallback is-warning", cfg.Message.Style)
	}
	if cfg.Message.Content != "" {
		t.Fatalf("message content = %q, want empty remote override", cfg.Message.Content)
	}
	if cfg.Message.Icon != "fas fa-bell" {
		t.Fatalf("message icon = %q, want fas fa-bell", cfg.Message.Icon)
	}
}

func TestResolveMessageMapsRemoteMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "",
			"value": "Remote content",
			"severity": "is-danger"
		}`))
	}))
	defer server.Close()

	assetsDir := t.TempDir()
	body := []byte(`
message:
  url: ` + server.URL + `
  mapping:
    title: id
    content: value
    style: severity
  style: is-warning
  title: Local title
  content: Local content
`)
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Message = ResolveMessage(context.Background(), cfg.Message)

	if cfg.Message.Title != "Local title" {
		t.Fatalf("message title = %q, want local title for empty mapped value", cfg.Message.Title)
	}
	if cfg.Message.Content != "Remote content" {
		t.Fatalf("message content = %q, want Remote content", cfg.Message.Content)
	}
	if cfg.Message.Style != "is-danger" {
		t.Fatalf("message style = %q, want is-danger", cfg.Message.Style)
	}
}

func TestResolveMessageKeepsLocalMessageWhenRemoteFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	assetsDir := t.TempDir()
	body := []byte(`
message:
  url: ` + server.URL + `
  refreshInterval: 10
  style: is-warning
  title: Local title
  content: Local content
`)
	if err := os.WriteFile(filepath.Join(assetsDir, "config.yml"), body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := (Loader{AssetsDir: assetsDir}).Load(context.Background(), "default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Message = ResolveMessage(context.Background(), cfg.Message)

	if cfg.Message.Title != "Local title" || cfg.Message.Content != "Local content" || cfg.Message.Style != "is-warning" {
		t.Fatalf("message = %+v, want local message", cfg.Message)
	}
	if cfg.Message.RefreshInterval != 10 {
		t.Fatalf("refreshInterval = %d, want parsed but unused value 10", cfg.Message.RefreshInterval)
	}
}
