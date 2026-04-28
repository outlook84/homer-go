package views

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"homer-go/internal/collectors"
	"homer-go/internal/config"
)

func TestNavLinkURLConvertsHomerHashPage(t *testing.T) {
	got := navLinkURL("#page2")

	if got != "/?page=page2" {
		t.Fatalf("navLinkURL() = %q, want /?page=page2", got)
	}
}

func TestNavLinkURLLeavesNonPageHashesAlone(t *testing.T) {
	tests := []string{"#", "##anchor", "#bad/page", "https://example.test/#page2"}

	for _, input := range tests {
		if got := navLinkURL(input); got != input {
			t.Fatalf("navLinkURL(%q) = %q, want unchanged", input, got)
		}
	}
}

func TestPageURLKeepsCurrentPageWithParams(t *testing.T) {
	got := pageURL("page2", url.Values{"layout": []string{"list"}, "search": []string{"status"}})

	if got != "/?layout=list&page=page2&search=status" {
		t.Fatalf("pageURL() = %q, want /?layout=list&page=page2&search=status", got)
	}
}

func TestPathsNormalizeAndPrefixInternalURLs(t *testing.T) {
	paths := NewPaths("dash/")

	if paths.BasePath != "/dash" {
		t.Fatalf("BasePath = %q, want /dash", paths.BasePath)
	}
	if got := paths.URL("/assets/homer-go.js"); got != "/dash/assets/homer-go.js" {
		t.Fatalf("URL() = %q, want /dash/assets/homer-go.js", got)
	}
	if got := paths.AssetURL("/assets/icons/logo.png"); got != "/dash/assets/icons/logo.png" {
		t.Fatalf("AssetURL() = %q, want /dash/assets/icons/logo.png", got)
	}
	if got := paths.AssetURL("/external/logo.png"); got != "/external/logo.png" {
		t.Fatalf("AssetURL() = %q, want /external/logo.png", got)
	}
	if got := paths.CookiePath(); got != "/dash/" {
		t.Fatalf("CookiePath() = %q, want /dash/", got)
	}
	if !paths.IsLocalURL("/dash/?page=page2") || paths.IsLocalURL("/other/") {
		t.Fatalf("IsLocalURL() did not constrain redirects to the base path")
	}
	if !paths.IsLocalURL("/dash?page=page2") {
		t.Fatalf("IsLocalURL() should allow base path with query")
	}
	for _, input := range []string{
		"https://example.test/",
		"//example.test/",
		"/\\example.test",
		"/other/?return=/dash/",
		"/dash/../other",
		"/dash/%2e%2e/other",
		"/dash%2f..%2fother",
		"/dash%2F..%2Fother",
		"/dash%5c..%5cother",
		"dashboard",
	} {
		if paths.IsLocalURL(input) {
			t.Fatalf("IsLocalURL(%q) = true, want false", input)
		}
		if got := paths.RedirectURL(input); got != "/dash/" {
			t.Fatalf("RedirectURL(%q) = %q, want /dash/", input, got)
		}
	}
	if got := paths.RedirectURL("/dash/?page=page2"); got != "/dash/?page=page2" {
		t.Fatalf("RedirectURL() = %q, want /dash/?page=page2", got)
	}
	if got := NewPaths("//example.test").BasePath; got != "" {
		t.Fatalf("NewPaths() accepted protocol-relative base path %q", got)
	}
}

func TestAssetURLUsesRegisteredLocalAssetResolver(t *testing.T) {
	paths := NewPaths("/dash")
	paths.AssetResolver = func(raw string) (string, bool) {
		if raw == "icons/logo.png" {
			return "/user-assets/test-token", true
		}
		return "", false
	}

	if got := paths.AssetURL("icons/logo.png"); got != "/dash/user-assets/test-token" {
		t.Fatalf("AssetURL() = %q, want mapped local asset URL", got)
	}
	if got := paths.AssetURL("/assets/icons/logo.png"); got != "/dash/assets/icons/logo.png" {
		t.Fatalf("AssetURL() = %q, want public assets URL", got)
	}
	if got := paths.AssetURL("/external/logo.png"); got != "/external/logo.png" {
		t.Fatalf("AssetURL() = %q, want unchanged URL", got)
	}
}

func TestPageURLAppliesBasePath(t *testing.T) {
	got := pageURLWithPaths("page2", url.Values{"search": []string{"status"}}, NewPaths("/dash"))

	if got != "/dash/?page=page2&search=status" {
		t.Fatalf("pageURLWithPaths() = %q, want /dash/?page=page2&search=status", got)
	}
}

func TestRenderSearchKeepsCurrentPage(t *testing.T) {
	var b strings.Builder

	renderSearch(&b, "page2", "status", "search", "desktop-search")
	html := b.String()

	if !strings.Contains(html, `name="page" value="page2"`) {
		t.Fatalf("renderSearch() missing hidden page input: %s", html)
	}
	if !strings.Contains(html, `name="search" value="status"`) {
		t.Fatalf("renderSearch() missing search input name: %s", html)
	}
	if !strings.Contains(html, `href="/?page=page2"`) {
		t.Fatalf("renderSearch() missing page-aware clear link: %s", html)
	}
}

func TestRenderPreferenceLinksRefreshReloadsCurrentPage(t *testing.T) {
	var b strings.Builder

	renderPreferenceLinks(&b, "page2", "status", Preferences{Theme: "auto", Layout: "columns"})
	html := b.String()

	if !strings.Contains(html, `class="navbar-item is-inline-block-mobile icon-button refresh-button" title="Refresh" href="/?page=page2&amp;search=status"`) {
		t.Fatalf("renderPreferenceLinks() missing page reload refresh link: %s", html)
	}
	if strings.Contains(html, "refresh=1") {
		t.Fatalf("renderPreferenceLinks() should not use refresh=1: %s", html)
	}
}

func TestRenderGroupsShowsNoResultsForEmptyFilter(t *testing.T) {
	var b strings.Builder
	cfg := config.Config{
		Services: []config.Group{{
			Name:  "missing",
			Icon:  "fas fa-search",
			Items: nil,
		}},
	}

	renderGroups(&b, cfg, Preferences{}, nil)
	html := b.String()

	if !strings.Contains(html, `class="empty-state"`) || !strings.Contains(html, `No results`) {
		t.Fatalf("renderGroups() missing empty state: %s", html)
	}
}

func TestRenderGroupsInheritsGroupClassForItems(t *testing.T) {
	var b strings.Builder
	cfg := config.Config{
		Columns: "3",
		Services: []config.Group{{
			Name:  "Apps",
			Class: "highlight-red",
			Items: []config.Item{{
				Name: "Calendar",
				URL:  "#",
			}},
		}},
	}

	renderGroups(&b, cfg, Preferences{}, nil)
	html := b.String()

	if !strings.Contains(html, `service-card-wrapper highlight-red`) {
		t.Fatalf("renderGroups() did not inherit group class: %s", html)
	}
}

func TestRenderFooterMatchesHomerContentWrapper(t *testing.T) {
	var b strings.Builder
	cfg := config.Config{Footer: `<p>Footer</p>`}

	renderFooter(&b, cfg)
	html := b.String()

	want := `<footer class="footer"><div class="container"><div class="content has-text-centered"><p>Footer</p></div></div></footer>`
	if html != want {
		t.Fatalf("renderFooter() = %q, want %q", html, want)
	}
}

func TestBaseCSSKeepsFixedFooterAboveCardLayers(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("..", "..", "assets", "css", "base.css"))
	if err != nil {
		t.Fatalf("read base.css: %v", err)
	}

	html := string(css)
	start := strings.Index(html, ".footer {")
	if start == -1 {
		t.Fatal("base.css missing .footer rule")
	}
	footer := html[start:]
	end := strings.Index(footer, "}")
	if end == -1 {
		t.Fatal("base.css has unterminated .footer rule")
	}
	footer = footer[:end]

	if !strings.Contains(footer, "position: fixed;") {
		t.Fatalf("footer CSS missing fixed positioning: %s", footer)
	}
	if !strings.Contains(footer, "z-index: 10;") {
		t.Fatalf("footer CSS must define a z-index above card child layers: %s", footer)
	}
}

func TestRenderItemRendersStatusNotifications(t *testing.T) {
	var b strings.Builder
	item := config.Item{Name: "Downloads", URL: "#"}
	status := collectors.Status{
		Badges: []collectors.Badge{{
			Label:  "Activity",
			Value:  "3",
			State:  "activity",
			Tone:   "info",
			Detail: "Active downloads",
		}},
	}

	renderItem(&b, item, "", status, "columns", "3")
	html := b.String()

	for _, want := range []string{
		`class="status-notifs"`,
		`class="status-notif activity is-info" title="Active downloads"`,
		`class="status-notif-label">Activity</span>3`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("renderItem() missing %s: %s", want, html)
		}
	}
}

func TestRenderItemRendersStatusIndicator(t *testing.T) {
	var b strings.Builder
	item := config.Item{Name: "Gatus", URL: "#"}
	status := collectors.Status{
		State:     "warn",
		Tone:      "warning",
		Detail:    "50%",
		Indicator: "50%",
	}

	renderItem(&b, item, "", status, "columns", "3")
	html := b.String()

	for _, want := range []string{
		`class="status-indicator warn is-warning" title="50%">50%</div>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("renderItem() missing %s: %s", want, html)
		}
	}
}

func TestRenderItemUsesStatusURLOverride(t *testing.T) {
	var b strings.Builder
	item := config.Item{
		Name: "Status",
		URL:  "https://status.example",
	}

	renderItem(&b, item, "", collectors.Status{URL: "https://status.example/status/public"}, "columns", "3")
	html := b.String()

	if !strings.Contains(html, `href="https://status.example/status/public"`) {
		t.Fatalf("renderItem() should prefer status URL override: %s", html)
	}
}

func TestRenderItemRendersQuickLinksOutsideCardLink(t *testing.T) {
	var b strings.Builder
	item := config.Item{
		Name: "Quick",
		URL:  "https://example.test",
		Quick: []config.QuickLink{{
			Name: "Docs",
			URL:  "https://example.test/docs",
		}},
	}

	renderItem(&b, item, "", collectors.Status{}, "columns", "3")
	html := b.String()

	if !strings.Contains(html, `<a class="card-link" href="https://example.test" rel="noreferrer" aria-label="Quick"></a><div class="card-content">`) {
		t.Fatalf("renderItem() should close card link before card content: %s", html)
	}
	if !strings.Contains(html, `<p class="quicklinks"><a href="https://example.test/docs" rel="noreferrer">Docs</a></p>`) {
		t.Fatalf("renderItem() missing quicklink: %s", html)
	}
	if opens, closes := strings.Count(html, "<div"), strings.Count(html, "</div>"); opens != closes {
		t.Fatalf("renderItem() divs are unbalanced: opens=%d closes=%d html=%s", opens, closes, html)
	}
}

func TestRenderItemEscapesSubtitleText(t *testing.T) {
	var b strings.Builder
	item := config.Item{Name: "Mixed", URL: "#", Subtitle: "服务 10.00% blocked & ok"}

	renderItem(&b, item, "", collectors.Status{}, "columns", "3")
	html := b.String()

	want := `<p class="subtitle">服务 10.00% blocked &amp; ok</p>`
	if !strings.Contains(html, want) {
		t.Fatalf("renderItem() subtitle = %s, want %s", html, want)
	}
}

func TestRenderItemRendersDotIndicatorWithTone(t *testing.T) {
	var b strings.Builder
	item := config.Item{Name: "Site", URL: "#"}
	status := collectors.Status{
		State:  "online",
		Tone:   "success",
		Detail: "204 No Content",
	}

	renderItem(&b, item, "", status, "columns", "3")
	html := b.String()

	if !strings.Contains(html, `class="status-indicator is-dot online is-success" title="204 No Content"></div>`) {
		t.Fatalf("renderItem() missing dot indicator tone: %s", html)
	}
	if strings.Contains(html, `class="status online"`) {
		t.Fatalf("renderItem() should not render legacy status class: %s", html)
	}
	if strings.Contains(html, `class="status-dot`) {
		t.Fatalf("renderItem() should not render legacy status dot: %s", html)
	}
}

func TestToneClassUsesExplicitTone(t *testing.T) {
	if got := toneClass("danger"); got != "is-danger" {
		t.Fatalf("toneClass() = %q, want is-danger", got)
	}
	if got := toneClass(""); got != "is-neutral" {
		t.Fatalf("toneClass() = %q, want is-neutral", got)
	}
}

func TestRenderItemDoesNotInferToneFromState(t *testing.T) {
	var b strings.Builder
	item := config.Item{Name: "Site", URL: "#"}
	status := collectors.Status{State: "online"}

	renderItem(&b, item, "", status, "columns", "3")
	html := b.String()

	if !strings.Contains(html, `class="status-indicator is-dot online is-neutral"`) {
		t.Fatalf("renderItem() should render state-only statuses as neutral: %s", html)
	}
	if strings.Contains(html, `class="status-indicator is-dot online is-success"`) {
		t.Fatalf("renderItem() should not infer success tone from state: %s", html)
	}
}

func TestRenderStylesheetsIncludesThemeAndVendorFiles(t *testing.T) {
	var b strings.Builder

	renderStylesheets(&b)
	html := b.String()

	for _, sheet := range []string{
		"/assets/vendor/bulma/css/bulma.min.css",
		"/assets/css/webfonts.css",
		"/assets/css/base.css",
		"/assets/css/status.css",
		"/assets/css/highlights.css",
		"/assets/css/themes/classic.css",
		"/assets/css/themes/neon.css",
		"/assets/css/themes/walkxcode.css",
		"/assets/homer-go.css",
	} {
		if !strings.Contains(html, `href="`+sheet+`"`) {
			t.Fatalf("renderStylesheets() missing %s: %s", sheet, html)
		}
	}
}

func TestDashboardIncludesPWAHeadLinks(t *testing.T) {
	var b strings.Builder
	component := Dashboard("Test", config.Config{Theme: "default", Header: true}, "default", "", Preferences{}, nil)

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	html := b.String()
	for _, item := range []string{
		`<meta name="theme-color" content="#3367d6">`,
		`<link rel="icon" type="image/png" sizes="32x32" href="/assets/icons/favicon-32x32.png">`,
		`<link rel="icon" type="image/png" sizes="16x16" href="/assets/icons/favicon-16x16.png">`,
		`<link rel="apple-touch-icon" sizes="180x180" href="/assets/icons/apple-touch-icon.png">`,
		`<link rel="manifest" href="/assets/manifest.json" crossorigin="use-credentials">`,
	} {
		if !strings.Contains(html, item) {
			t.Fatalf("Dashboard() missing %s: %s", item, html)
		}
	}
}

func TestDashboardExposesCurrentPageToRuntimeConfig(t *testing.T) {
	var b strings.Builder
	component := Dashboard("Test", config.Config{Theme: "default", Header: true}, "page2", "", Preferences{}, nil)

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(b.String(), `window.HOMER_PAGE="page2";`) {
		t.Fatalf("Dashboard() missing HOMER_PAGE runtime config: %s", b.String())
	}
}

func TestDashboardRegistersServiceWorkerWithAutoReload(t *testing.T) {
	var b strings.Builder
	component := Dashboard("Test", config.Config{Theme: "default", Header: true}, "default", "", Preferences{}, nil)

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	html := b.String()
	if !strings.Contains(html, `navigator.serviceWorker.register("/sw.js")`) ||
		!strings.Contains(html, `controllerchange`) ||
		!strings.Contains(html, `window.location.reload()`) {
		t.Fatalf("Dashboard() missing service worker registration/update reload: %s", html)
	}
}

func TestDashboardAppliesBasePathToInternalAssetsAndActions(t *testing.T) {
	var b strings.Builder
	component := DashboardWithPaths("Test", config.Config{
		Theme:      "default",
		Header:     true,
		Logo:       "/assets/icons/homer-go-logo-v2.png",
		Stylesheet: []string{"/assets/custom.css"},
		Colors: map[string]map[string]string{
			"custom": {"background-image": "/assets/bg.png"},
		},
		Services: []config.Group{{
			Name: "Apps",
			Logo: "/assets/icons/group.png",
			Items: []config.Item{{
				Name: "Calendar",
				URL:  "#",
				Logo: "/assets/icons/item.png",
			}},
		}},
	}, "page2", "status", Preferences{Theme: "auto", Layout: "columns"}, nil, NewPaths("/dash"))

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	html := b.String()
	for _, item := range []string{
		`href="/dash/assets/icons/favicon-32x32.png"`,
		`href="/dash/assets/homer-go.css"`,
		`href="/dash/assets/custom.css"`,
		`src="/dash/assets/icons/homer-go-logo-v2.png"`,
		`url(/dash/assets/bg.png)`,
		`src="/dash/assets/icons/group.png"`,
		`src="/dash/assets/icons/item.png"`,
		`href="/dash/?page=page2&amp;search=status"`,
		`href="/dash/theme?value=light&amp;return=%2Fdash%2F%3Fpage%3Dpage2%26search%3Dstatus"`,
		`action="/dash/"`,
		`window.HOMER_BASE_PATH="/dash";`,
		`navigator.serviceWorker.register("/dash/sw.js")`,
		`src="/dash/assets/homer-go.js"`,
	} {
		if !strings.Contains(html, item) {
			t.Fatalf("DashboardWithPaths() missing %s: %s", item, html)
		}
	}
}

func TestServicesFragmentAppliesBasePathToLocalAssetLogos(t *testing.T) {
	var b strings.Builder
	component := ServicesFragmentWithPaths(config.Config{
		Services: []config.Group{{
			Name: "Apps",
			Logo: "/assets/icons/group.png",
			Items: []config.Item{{
				Name: "Calendar",
				URL:  "#",
				Logo: "/assets/icons/item.png",
			}},
		}},
	}, Preferences{}, nil, NewPaths("/dash"))

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	html := b.String()
	for _, item := range []string{
		`src="/dash/assets/icons/group.png"`,
		`src="/dash/assets/icons/item.png"`,
	} {
		if !strings.Contains(html, item) {
			t.Fatalf("ServicesFragmentWithPaths() missing %s: %s", item, html)
		}
	}
}

func TestDashboardRendersOfflineMessageWhenConnectivityCheckEnabled(t *testing.T) {
	var b strings.Builder
	component := Dashboard("Test", config.Config{Theme: "default", Header: true, ConnectivityCheck: true}, "default", "", Preferences{}, nil)

	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	html := b.String()
	if !strings.Contains(html, `data-offline-message`) ||
		!strings.Contains(html, `window.HOMER_CONNECTIVITY_CHECK=true;`) {
		t.Fatalf("Dashboard() missing connectivity check markup/config: %s", html)
	}
}

func TestRuntimeScriptPropagatesPageToFragments(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "assets", "homer-go.js"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(body)

	if !strings.Contains(script, `window.HOMER_PAGE !== "default"`) ||
		!strings.Contains(script, `url.searchParams.set("page", window.HOMER_PAGE)`) {
		t.Fatalf("homer-go.js does not propagate HOMER_PAGE to fragment URLs")
	}
}

func TestRuntimeScriptCleansConnectivityTimestampAndPausesRefreshOffline(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "assets", "homer-go.js"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(body)

	if !strings.Contains(script, `searchParams.delete("t")`) ||
		!strings.Contains(script, `servicesRefreshing || document.hidden || offline`) ||
		!strings.Contains(script, `messageRefreshing || document.hidden || offline`) {
		t.Fatalf("homer-go.js does not clean t or pause refresh while offline")
	}
}

func TestRuntimeScriptIgnoresStaleConnectivityChecks(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "assets", "homer-go.js"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(body)

	for _, want := range []string{
		`var connectivityCheckID = 0;`,
		`var checkID = ++connectivityCheckID;`,
		`if (checkID === connectivityCheckID) setOffline(!response.ok);`,
		`if (checkID === connectivityCheckID) setOffline(true);`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("homer-go.js missing stale connectivity check guard %s", want)
		}
	}
}

func TestRuntimeScriptPollsForRecoveryOnlyWhileOffline(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "assets", "homer-go.js"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(body)

	for _, want := range []string{
		`var offlineRecoveryTimer = 0;`,
		`var offlineRecoveryIntervalMs = 5000;`,
		`syncOfflineRecoveryTimer();`,
		`if (!offline || document.hidden)`,
		`window.clearTimeout(offlineRecoveryTimer);`,
		`offlineRecoveryTimer = window.setTimeout(function () {`,
		`offlineRecoveryTimer = 0;`,
		`checkOffline().finally(syncOfflineRecoveryTimer);`,
		`}, offlineRecoveryIntervalMs);`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("homer-go.js missing offline recovery polling code %s", want)
		}
	}
}
