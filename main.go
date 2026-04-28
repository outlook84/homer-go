package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"homer-go/internal/collectors"
	"homer-go/internal/config"
	"homer-go/internal/views"

	"github.com/a-h/templ"
)

//go:embed assets
var embeddedFiles embed.FS

var buildID = "dev"

func main() {
	addr := flag.String("addr", env("HOMER_GO_ADDR", ":8732"), "HTTP listen address")
	assetsDir := flag.String("assets", env("HOMER_GO_ASSETS_DIR", "assets"), "assets directory")
	dataDir := flag.String("data", env("HOMER_GO_DATA_DIR", "."), "data directory containing config.yml and user assets")
	basePath := flag.String("base-path", env("HOMER_GO_BASE_PATH", ""), "public URL path prefix, e.g. /homer-go")
	flag.Parse()
	localAssets := newLocalAssetRegistry(*dataDir)
	paths := views.NewPaths(*basePath)
	paths.AssetResolver = localAssets.Resolve
	assetFS := assetFileSystem(*assetsDir)
	fingerprintFS := assetFingerprintFS(*assetsDir)
	exampleConfig, _ := embeddedFiles.ReadFile("assets/config.yml")

	loader := config.Loader{
		AssetsDir:     *assetsDir,
		ConfigDir:     *dataDir,
		ExampleConfig: exampleConfig,
		AutoInit:      true,
		OnInit: func(path string) {
			log.Printf("config.yml not found; generated example config at %s", abs(path))
		},
	}
	registry := collectors.NewRegistry()
	registry.Register(collectors.AdGuardHome{})
	registry.Register(collectors.Docuseal{})
	registry.Register(collectors.Gitea{})
	registry.Register(collectors.Emby{})
	registry.Register(collectors.FreshRSS{})
	registry.Register(collectors.Glances{})
	registry.Register(collectors.Gotify{})
	registry.Register(collectors.Healthchecks{})
	registry.Register(collectors.HomeAssistant{})
	registry.Register(collectors.HyperHDR{})
	registry.Register(collectors.DockerSocketProxy{})
	registry.Register(collectors.Gatus{})
	registry.Register(collectors.Immich{})
	registry.Register(collectors.Jellyfin{})
	registry.Register(collectors.Lidarr{})
	registry.Register(collectors.Matrix{})
	registry.Register(collectors.Mealie{})
	registry.Register(collectors.Medusa{})
	registry.Register(collectors.Miniflux{})
	registry.Register(collectors.Mylar{})
	registry.Register(collectors.NetAlertx{})
	registry.Register(collectors.Nextcloud{})
	registry.Register(collectors.OpenHAB{})
	registry.Register(collectors.Olivetin{})
	registry.Register(collectors.PaperlessNG{})
	registry.Register(collectors.PeaNUT{})
	registry.Register(collectors.PiAlert{})
	registry.Register(collectors.Portainer{})
	registry.Register(collectors.Ping{})
	registry.Register(collectors.Proxmox{})
	registry.Register(collectors.Prometheus{})
	registry.Register(collectors.Prowlarr{})
	registry.Register(collectors.QBittorrent{})
	registry.Register(collectors.Radarr{})
	registry.Register(collectors.Readarr{})
	registry.Register(collectors.SABnzbd{})
	registry.Register(collectors.Scrutiny{})
	registry.Register(collectors.Sonarr{})
	registry.Register(collectors.SpeedtestTracker{})
	registry.Register(collectors.Tautulli{})
	registry.Register(collectors.Tdarr{})
	registry.Register(collectors.Traefik{})
	registry.Register(collectors.TruenasScale{})
	registry.Register(collectors.UptimeKuma{})
	registry.Register(collectors.Vaultwarden{})
	registry.Register(collectors.Wallabag{})
	registry.Register(collectors.WUD{})

	appMux := http.NewServeMux()
	appMux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(assetFS)))
	appMux.Handle("/user-assets/", http.StripPrefix("/user-assets/", localAssets))
	appMux.HandleFunc("/sw.js", serviceWorkerHandlerWithFS(buildID, fingerprintFS, paths))
	appMux.HandleFunc("/fragments/message", messageFragmentHandler(loader))
	appMux.HandleFunc("/fragments/services", servicesFragmentHandler(loader, registry, paths))
	appMux.HandleFunc("/theme", preferenceHandler("theme", []string{"auto", "light", "dark"}, paths))
	appMux.HandleFunc("/layout", preferenceHandler("layout", []string{"columns", "list"}, paths))
	appMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/page/") {
			http.NotFound(w, r)
			return
		}

		page, ok := pageFromRequest(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		cfg, err := loader.Load(r.Context(), page)
		if err != nil {
			renderConfigLoadErrorWithPaths(w, r, err, paths)
			return
		}

		prefs := views.Preferences{
			Theme:  cookieOrDefault(r, "theme", cfg.Defaults.ColorTheme, "auto"),
			Layout: cookieOrDefault(r, "layout", cfg.Defaults.Layout, "columns"),
		}
		query := strings.TrimSpace(r.URL.Query().Get("search"))
		renderCfg := cfg.Filter(query)

		title := cfg.DocumentTitle
		if title == "" {
			title = strings.Join(nonEmpty(cfg.Title, cfg.Subtitle), " | ")
		}
		setDynamicCacheHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templ.Handler(views.DashboardWithPaths(title, renderCfg, page, query, prefs, nil, paths)).ServeHTTP(w, r)
	})

	var mux http.Handler = appMux
	if paths.BasePath != "" {
		rootMux := http.NewServeMux()
		rootMux.Handle(paths.BasePath+"/", http.StripPrefix(paths.BasePath, appMux))
		rootMux.HandleFunc(paths.BasePath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, paths.BasePath+"/", http.StatusMovedPermanently)
		})
		mux = rootMux
	}

	if _, err := os.Stat(*assetsDir); errors.Is(err, os.ErrNotExist) {
		log.Printf("assets directory %q does not exist; embedded assets will be used", *assetsDir)
	}
	logUnsupportedConfig(loader, registry)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	log.Printf("homer-go listening on %s, data=%s, assets=%s, config=%s, basePath=%q", *addr, abs(*dataDir), abs(*assetsDir), abs(loaderConfigPath(loader)), paths.BasePath)
	if err := listenAndServe(ctx, server, 5*time.Second); err != nil {
		log.Fatal(err)
	}
}

func listenAndServe(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	return serveHTTP(ctx, server, listener, shutdownTimeout, closeDefaultHTTPClientIdleConnections)
}

func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration, closeIdleClientConnections func()) error {
	if closeIdleClientConnections == nil {
		closeIdleClientConnections = func() {}
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Printf("shutdown signal received; draining HTTP server for up to %s", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v; forcing HTTP server close", err)
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			log.Printf("forced HTTP server close failed: %v", closeErr)
		}
		closeIdleClientConnections()
		return err
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		closeIdleClientConnections()
		return err
	}
	closeIdleClientConnections()
	log.Printf("shutdown complete")
	return nil
}

func closeDefaultHTTPClientIdleConnections() {
	http.DefaultClient.CloseIdleConnections()
}

func assetFileSystem(assetsDir string) http.FileSystem {
	sub, err := fs.Sub(embeddedFiles, "assets")
	if err != nil {
		return http.Dir(assetsDir)
	}
	return overlayFileSystem{
		user:     http.Dir(assetsDir),
		embedded: http.FS(sub),
	}
}

func assetFingerprintFS(assetsDir string) fs.FS {
	sub, err := fs.Sub(embeddedFiles, "assets")
	if err != nil {
		return os.DirFS(assetsDir)
	}
	return overlayFS{
		user:     os.DirFS(assetsDir),
		embedded: sub,
	}
}

type overlayFileSystem struct {
	user     http.FileSystem
	embedded http.FileSystem
}

func (o overlayFileSystem) Open(name string) (http.File, error) {
	file, err := o.user.Open(name)
	if err == nil {
		return file, nil
	}
	return o.embedded.Open(name)
}

type overlayFS struct {
	user     fs.FS
	embedded fs.FS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	file, err := o.user.Open(name)
	if err == nil {
		return file, nil
	}
	return o.embedded.Open(name)
}

type localAssetRegistry struct {
	root   string
	secret []byte
	mu     sync.RWMutex
	files  map[string]string
}

func newLocalAssetRegistry(root string) *localAssetRegistry {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		sum := sha256.Sum256([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
		secret = sum[:]
	}
	return &localAssetRegistry{
		root:   filepath.Clean(absRoot),
		secret: secret,
		files:  map[string]string{},
	}
}

func (r *localAssetRegistry) Resolve(raw string) (string, bool) {
	path, ok := r.resolvePath(raw)
	if !ok {
		return "", false
	}
	token := r.token(path)
	r.mu.Lock()
	r.files[token] = path
	r.mu.Unlock()
	return "/user-assets/" + token, true
}

func (r *localAssetRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.Trim(req.URL.Path, "/")
	if token == "" || strings.Contains(token, "/") || strings.Contains(token, `\`) {
		http.NotFound(w, req)
		return
	}
	r.mu.RLock()
	path, ok := r.files[token]
	r.mu.RUnlock()
	if !ok {
		http.NotFound(w, req)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, req)
		return
	}
	setDynamicCacheHeaders(w)
	http.ServeContent(w, req, filepath.Base(path), info.ModTime(), file)
}

func (r *localAssetRegistry) resolvePath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "/assets/") || isExternalURL(raw) {
		return "", false
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, `\`) || filepath.IsAbs(raw) || isWindowsAbsPath(raw) {
		return "", false
	}
	name := filepath.FromSlash(strings.ReplaceAll(raw, "\\", "/"))
	abs, err := filepath.Abs(filepath.Join(r.root, name))
	if err != nil {
		return "", false
	}
	abs = filepath.Clean(abs)
	root, err := filepath.EvalSymlinks(r.root)
	if err != nil {
		return "", false
	}
	root = filepath.Clean(root)
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	realPath = filepath.Clean(realPath)
	rel, err := filepath.Rel(root, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if isConfigAssetPath(rel) {
		return "", false
	}
	return realPath, true
}

func (r *localAssetRegistry) token(path string) string {
	h := hmac.New(sha256.New, r.secret)
	_, _ = h.Write([]byte(filepath.Clean(path)))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func isExternalURL(raw string) bool {
	if strings.HasPrefix(raw, "//") {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme != "" && !isWindowsAbsPath(raw)
}

func isWindowsAbsPath(raw string) bool {
	return strings.HasPrefix(raw, `\\`) || len(raw) >= 3 && raw[1] == ':' && (raw[2] == '\\' || raw[2] == '/')
}

func isConfigAssetPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

func logUnsupportedConfig(loader config.Loader, registry *collectors.Registry) {
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	cfg, err := loader.Load(ctx, "default")
	if err != nil {
		return
	}
	for _, path := range config.UnsupportedConfigPaths(cfg) {
		log.Printf("unsupported config key %q is present but currently ignored", path)
	}
	for _, unsupported := range registry.UnsupportedCollectors(cfg) {
		log.Printf("unsupported collector type %q for service %q in group %q at services[%d].items[%d]; status collection will be skipped", unsupported.Type, unsupported.ItemName, unsupported.GroupName, unsupported.GroupIndex, unsupported.ItemIndex)
	}
}

func serviceWorkerHandler(id string, assetsDir string) http.HandlerFunc {
	return serviceWorkerHandlerWithPaths(id, assetsDir, views.Paths{})
}

func serviceWorkerHandlerWithPaths(id string, assetsDir string, paths views.Paths) http.HandlerFunc {
	return serviceWorkerHandlerWithFS(id, os.DirFS(assetsDir), paths)
}

func serviceWorkerHandlerWithFS(id string, assetsFS fs.FS, paths views.Paths) http.HandlerFunc {
	assets := []string{
		"/",
		"/assets/manifest.json",
		"/assets/homer-go.css",
		"/assets/homer-go.js",
		"/assets/vendor/bulma/css/bulma.min.css",
		"/assets/css/themes/classic.css",
		"/assets/css/webfonts.css",
		"/assets/css/base.css",
		"/assets/css/status.css",
		"/assets/css/highlights.css",
		"/assets/css/themes/neon.css",
		"/assets/css/themes/walkxcode.css",
		"/assets/webfonts/noto/noto-latin-normal.woff2",
		"/assets/vendor/fontawesome/css/all.min.css",
		"/assets/vendor/fontawesome/webfonts/fa-brands-400.woff2",
		"/assets/vendor/fontawesome/webfonts/fa-regular-400.woff2",
		"/assets/vendor/fontawesome/webfonts/fa-solid-900.woff2",
		"/assets/vendor/fontawesome/webfonts/fa-v4compatibility.woff2",
		"/assets/icons/homer-go-logo-v2.png",
		"/assets/icons/pwa-192x192.png",
		"/assets/icons/pwa-512x512.png",
		"/assets/icons/pwa-maskable-512x512.png",
		"/assets/icons/apple-touch-icon.png",
		"/assets/icons/favicon-32x32.png",
		"/assets/icons/favicon-16x16.png",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		writeServiceWorker(w, cacheNameWithFS(id, assetsFS, assets), paths, assets)
	}
}

func cacheName(id string, assetsDir string, assets []string) string {
	return cacheNameWithFS(id, os.DirFS(assetsDir), assets)
}

func cacheNameWithFS(id string, assetsFS fs.FS, assets []string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "dev"
	}
	return "homer-go-" + id + "-" + assetsFingerprint(assetsFS, assets)
}

func assetsFingerprint(assetsFS fs.FS, assets []string) string {
	h := sha256.New()
	for _, asset := range assets {
		if !strings.HasPrefix(asset, "/assets/") {
			continue
		}
		name := strings.TrimPrefix(asset, "/assets/")
		if _, err := io.WriteString(h, asset+"\n"); err != nil {
			continue
		}
		f, err := assetsFS.Open(filepath.ToSlash(name))
		if err != nil {
			_, _ = io.WriteString(h, "missing\n")
			continue
		}
		_, _ = io.Copy(h, f)
		_ = f.Close()
		_, _ = io.WriteString(h, "\n")
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return sum[:12]
}

func setDynamicCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
}

func writeServiceWorker(w http.ResponseWriter, cacheName string, paths views.Paths, assets []string) {
	var b strings.Builder
	const cachePrefix = "homer-go-"
	b.WriteString("const CACHE_PREFIX = ")
	b.WriteString(strconv.Quote(cachePrefix))
	b.WriteString(";\n")
	b.WriteString("const CACHE = ")
	b.WriteString(strconv.Quote(cacheName))
	b.WriteString(";\nconst ASSETS = [")
	for i, asset := range assets {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(paths.URL(asset)))
	}
	b.WriteString(`];
const BASE_PATH = `)
	b.WriteString(strconv.Quote(paths.BasePath))
	b.WriteString(`;

self.addEventListener("install", event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(ASSETS)));
  self.skipWaiting();
});

self.addEventListener("activate", event => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(key => key.startsWith(CACHE_PREFIX) && key !== CACHE).map(key => caches.delete(key)))
    )
  );
  self.clients.claim();
});

function cacheResponse(request, response) {
  if (!response.ok || response.type !== "basic") return Promise.resolve();
  return caches.open(CACHE).then(cache => cache.put(request, response.clone()));
}

function fetchAndCache(event) {
  const responsePromise = fetch(event.request);
  event.waitUntil(
    responsePromise
      .then(response => cacheResponse(event.request, response))
      .catch(() => {})
  );
  return responsePromise;
}

function fetchNavigation(event) {
  const responsePromise = fetch(event.request, { cache: "reload" });
  event.waitUntil(
    responsePromise
      .then(response => cacheResponse(event.request, response))
      .catch(() => {})
  );
  return responsePromise;
}

self.addEventListener("fetch", event => {
  if (event.request.method !== "GET") return;

  const url = new URL(event.request.url);
  if (url.origin !== self.location.origin) return;
  if (BASE_PATH && url.pathname !== BASE_PATH && !url.pathname.startsWith(BASE_PATH + "/")) return;
  const appPath = BASE_PATH ? url.pathname.slice(BASE_PATH.length) || "/" : url.pathname;
  if (appPath === "/sw.js" || appPath.startsWith("/fragments/") || appPath === "/theme" || appPath === "/layout") return;
  const precached = ASSETS.includes(url.pathname);

  if (event.request.mode === "navigate") {
    event.respondWith(
      fetchNavigation(event)
        .catch(() => caches.match(event.request).then(cached => cached || caches.match(BASE_PATH + "/")))
    );
    return;
  }

  if (appPath.startsWith("/assets/")) {
    if (!precached) {
      event.respondWith(
        fetchAndCache(event).catch(() => caches.match(event.request))
      );
      return;
    }

    event.respondWith(
      caches.match(event.request).then(cached =>
        cached || fetchAndCache(event)
      )
    );
  }
});
`)
	_, _ = w.Write([]byte(b.String()))
}

func renderConfigLoadError(w http.ResponseWriter, r *http.Request, err error) {
	renderConfigLoadErrorWithPaths(w, r, err, views.Paths{})
}

func renderConfigLoadErrorWithPaths(w http.ResponseWriter, r *http.Request, err error, paths views.Paths) {
	var loadErr *config.LoadError
	if !errors.As(err, &loadErr) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	setDynamicCacheHeaders(w)
	switch loadErr.Kind {
	case config.ErrorNotFound:
		w.WriteHeader(http.StatusOK)
		templ.Handler(views.GetStartedPageWithPaths(paths)).ServeHTTP(w, r)
	case config.ErrorPageNotFound:
		http.NotFound(w, r)
	case config.ErrorConfigDir:
		w.WriteHeader(http.StatusInternalServerError)
		templ.Handler(views.ConfigErrorPageWithPaths("Error loading configuration", loadErr.Error(), paths)).ServeHTTP(w, r)
	case config.ErrorExternal:
		w.WriteHeader(http.StatusBadGateway)
		templ.Handler(views.ConfigErrorPageWithPaths("Error loading external configuration", loadErr.Error(), paths)).ServeHTTP(w, r)
	case config.ErrorParse:
		w.WriteHeader(http.StatusInternalServerError)
		templ.Handler(views.ConfigErrorPageWithPaths("Error parsing configuration", loadErr.Error(), paths)).ServeHTTP(w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		templ.Handler(views.ConfigErrorPageWithPaths("Error loading configuration", loadErr.Error(), paths)).ServeHTTP(w, r)
	}
}

func messageFragmentHandler(loader config.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, ok := pageFromRequest(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		cfg, err := loader.Load(r.Context(), page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		msg := config.ResolveMessage(r.Context(), cfg.Message)
		setDynamicCacheHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templ.Handler(views.MessageFragment(msg)).ServeHTTP(w, r)
	}
}

func servicesFragmentHandler(loader config.Loader, registry *collectors.Registry, paths views.Paths) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, ok := pageFromRequest(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		cfg, err := loader.Load(r.Context(), page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		prefs := views.Preferences{
			Theme:  cookieOrDefault(r, "theme", cfg.Defaults.ColorTheme, "auto"),
			Layout: cookieOrDefault(r, "layout", cfg.Defaults.Layout, "columns"),
		}
		query := strings.TrimSpace(r.URL.Query().Get("search"))
		renderCfg := cfg.Filter(query)
		statuses := registry.Collect(r.Context(), renderCfg, 2500*time.Millisecond)
		setDynamicCacheHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templ.Handler(views.ServicesFragmentWithPaths(renderCfg, prefs, statuses, paths)).ServeHTTP(w, r)
	}
}

func preferenceHandler(name string, allowed []string, paths views.Paths) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value := r.URL.Query().Get("value")
		if !contains(allowed, value) {
			http.Error(w, "invalid preference", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     paths.CookiePath(),
			SameSite: http.SameSiteLaxMode,
			Secure:   secureCookie(r),
			MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		})
		http.Redirect(w, r, paths.RedirectURL(r.URL.Query().Get("return")), http.StatusSeeOther)
	}
}

func secureCookie(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func pageFromRequest(r *http.Request) (string, bool) {
	page := strings.TrimSpace(r.URL.Query().Get("page"))
	if page == "" && strings.HasPrefix(r.URL.Path, "/page/") {
		page, _ = url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/page/"))
	}
	return normalizePage(page)
}

func normalizePage(page string) (string, bool) {
	page = strings.TrimSpace(page)
	if page == "" {
		return "default", true
	}
	if page == "default" {
		return page, true
	}
	if !config.ValidPageName(page) {
		return "", false
	}
	return page, true
}

func cookieOrDefault(r *http.Request, name, fallback, zero string) string {
	if c, err := r.Cookie(name); err == nil && c.Value != "" {
		return c.Value
	}
	if fallback != "" {
		return fallback
	}
	return zero
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func abs(path string) string {
	out, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return out
}

func loaderConfigPath(loader config.Loader) string {
	dir := loader.ConfigDir
	if dir == "" {
		dir = loader.AssetsDir
	}
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "config.yml")
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
