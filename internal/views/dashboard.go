package views

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/url"
	"strconv"
	"strings"

	"homer-go/internal/collectors"
	"homer-go/internal/config"

	"github.com/a-h/templ"
)

type Preferences struct {
	Theme  string
	Layout string
}

type AssetResolver func(string) (string, bool)

type Paths struct {
	BasePath      string
	AssetResolver AssetResolver
}

func NewPaths(basePath string) Paths {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || basePath == "/" {
		return Paths{}
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimRight(basePath, "/")
	if strings.Contains(basePath, "//") || strings.Contains(basePath, "?") || strings.Contains(basePath, "#") {
		return Paths{}
	}
	return Paths{BasePath: basePath}
}

func (p Paths) URL(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if p.BasePath == "" {
		return path
	}
	if path == "/" {
		return p.BasePath + "/"
	}
	return p.BasePath + path
}

func (p Paths) AssetURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, "/assets/") {
		return p.URL(raw)
	}
	if p.AssetResolver != nil {
		if path, ok := p.AssetResolver(raw); ok {
			return p.URL(path)
		}
	}
	return raw
}

func (p Paths) CookiePath() string {
	return p.URL("/")
}

func (p Paths) IsLocalURL(raw string) bool {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return false
	}
	if p.BasePath == "" {
		return true
	}
	return raw == p.BasePath || strings.HasPrefix(raw, p.BasePath+"/")
}

func GetStartedPage() templ.Component {
	return GetStartedPageWithPaths(Paths{})
}

func GetStartedPageWithPaths(paths Paths) templ.Component {
	return basicPage("No configuration found", paths, func(w io.Writer) {
		write(w, "<article class=\"setup-panel\"><div class=\"setup-content\">")
		write(w, "<p class=\"setup-title\">No configuration found!</p>")
		write(w, "<p>Check out the documentation to start building your dashboard.</p>")
		write(w, "<p><a class=\"setup-button\" href=\"https://github.com/bastienwirtz/homer/blob/main/docs/configuration.md#configuration\" target=\"_blank\" rel=\"noreferrer\">Get started &rarr;</a></p>")
		write(w, "</div></article>")
	})
}

func ConfigErrorPage(title string, message string) templ.Component {
	return ConfigErrorPageWithPaths(title, message, Paths{})
}

func ConfigErrorPageWithPaths(title string, message string, paths Paths) templ.Component {
	return basicPage(title, paths, func(w io.Writer) {
		write(w, "<article class=\"message is-danger config-error\">")
		write(w, "<div class=\"message-header\"><p><i class=\"fa-fw fas fa-triangle-exclamation\"></i>"+esc(title)+"</p></div>")
		write(w, "<div class=\"message-body\"><pre>"+esc(message)+"</pre></div>")
		write(w, "</article>")
	})
}

func basicPage(title string, paths Paths, body func(io.Writer)) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		write(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
		write(w, "<title>"+esc(title)+"</title>")
		renderPWAHeadWithPaths(w, paths)
		renderStylesheetsWithPaths(w, paths)
		write(w, "</head><body><div id=\"app\" class=\"theme-default page-default auto no-footer\">")
		write(w, "<main id=\"main-section\" class=\"section\"><div class=\"container\">")
		body(w)
		write(w, "</div></main>")
		write(w, "</div></body></html>")
		return nil
	})
}

func Dashboard(title string, cfg config.Config, page string, query string, prefs Preferences, statuses map[string]collectors.Status) templ.Component {
	return DashboardWithPaths(title, cfg, page, query, prefs, statuses, Paths{})
}

func DashboardWithPaths(title string, cfg config.Config, page string, query string, prefs Preferences, statuses map[string]collectors.Status, paths Paths) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		write(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
		write(w, "<title>"+esc(title)+"</title>")
		renderPWAHeadWithPaths(w, paths)
		renderStylesheetsWithPaths(w, paths)
		for _, sheet := range cfg.Stylesheet {
			write(w, "<link rel=\"stylesheet\" href=\""+attr(paths.AssetURL(sheet))+"\">")
		}
		writeDynamicTheme(w, cfg, paths)
		write(w, "</head>")
		appClass := fmt.Sprintf("theme-%s page-%s %s %s", cfg.Theme, pageClass(page), themeClass(prefs.Theme), layoutClass(prefs.Layout))
		if footerHidden(cfg.Footer) {
			appClass += " no-footer"
		}
		write(w, "<body><div id=\"app\" class=\""+attr(appClass)+"\">")
		write(w, "<div id=\"bighead\">")
		renderHeader(w, cfg, paths)
		renderNav(w, cfg, page, query, prefs, paths)
		write(w, "</div>")
		write(w, "<main id=\"main-section\" class=\"section\"><div class=\"container\">")
		renderOfflineMessage(w, cfg)
		write(w, "<div data-online-content>")
		renderMessageFragment(w, cfg.Message)
		renderServicesFragmentWithPaths(w, cfg, prefs, statuses, paths)
		write(w, "</div>")
		write(w, "</div></main>")
		renderFooter(w, cfg)
		renderRuntimeConfig(w, cfg, page, paths)
		write(w, "</div></body></html>")
		return nil
	})
}

func renderPWAHead(w io.Writer) {
	renderPWAHeadWithPaths(w, Paths{})
}

func renderPWAHeadWithPaths(w io.Writer, paths Paths) {
	write(w, "<meta name=\"theme-color\" content=\"#3367d6\">")
	write(w, "<link rel=\"icon\" type=\"image/png\" sizes=\"32x32\" href=\""+attr(paths.URL("/assets/icons/favicon-32x32.png"))+"\">")
	write(w, "<link rel=\"icon\" type=\"image/png\" sizes=\"16x16\" href=\""+attr(paths.URL("/assets/icons/favicon-16x16.png"))+"\">")
	write(w, "<link rel=\"apple-touch-icon\" sizes=\"180x180\" href=\""+attr(paths.URL("/assets/icons/apple-touch-icon.png"))+"\">")
	write(w, "<link rel=\"manifest\" href=\""+attr(paths.URL("/assets/manifest.json"))+"\" crossorigin=\"use-credentials\">")
}

func renderStylesheets(w io.Writer) {
	renderStylesheetsWithPaths(w, Paths{})
}

func renderStylesheetsWithPaths(w io.Writer, paths Paths) {
	for _, sheet := range []string{
		"/assets/vendor/fontawesome/css/all.min.css",
		"/assets/vendor/bulma/css/bulma.min.css",
		"/assets/css/themes/classic.css",
		"/assets/css/webfonts.css",
		"/assets/css/base.css",
		"/assets/css/status.css",
		"/assets/css/highlights.css",
		"/assets/css/themes/neon.css",
		"/assets/css/themes/walkxcode.css",
		"/assets/homer-go.css",
	} {
		write(w, "<link rel=\"stylesheet\" href=\""+attr(paths.URL(sheet))+"\">")
	}
}

func MessageFragment(msg config.Message) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		renderMessageFragment(w, msg)
		return nil
	})
}

func ServicesFragment(cfg config.Config, prefs Preferences, statuses map[string]collectors.Status) templ.Component {
	return ServicesFragmentWithPaths(cfg, prefs, statuses, Paths{})
}

func ServicesFragmentWithPaths(cfg config.Config, prefs Preferences, statuses map[string]collectors.Status, paths Paths) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		renderServicesFragmentWithPaths(w, cfg, prefs, statuses, paths)
		return nil
	})
}

func renderHeader(w io.Writer, cfg config.Config, paths Paths) {
	if !cfg.Header {
		return
	}
	write(w, "<section class=\"first-line\"><div class=\"container header-inner\">")
	write(w, "<a class=\"logo\" href=\""+attr(paths.URL("/"))+"\">")
	if cfg.Logo != "" {
		write(w, "<img src=\""+attr(paths.AssetURL(cfg.Logo))+"\" alt=\"dashboard logo\">")
	}
	if cfg.Icon != "" {
		write(w, "<i class=\""+attr(cfg.Icon)+"\"></i>")
	}
	write(w, "</a><div class=\"dashboard-title\">")
	if cfg.Subtitle != "" {
		write(w, "<span class=\"headline\">"+esc(cfg.Subtitle)+"</span>")
	}
	write(w, "<h1>"+esc(cfg.Title)+"</h1></div></div></section>")
}

func renderNav(w io.Writer, cfg config.Config, page string, query string, prefs Preferences, paths Paths) {
	write(w, "<div class=\"container-fluid\"><nav class=\"navbar\" role=\"navigation\" aria-label=\"main navigation\"><div class=\"container nav-inner\"><input id=\"nav-closed\" name=\"nav-state\" class=\"nav-state nav-state-closed\" type=\"radio\" checked><input id=\"navbar-toggle\" name=\"nav-state\" class=\"nav-state navbar-toggle\" type=\"radio\"><input id=\"search-toggle\" name=\"nav-state\" class=\"nav-state search-toggle\" type=\"radio\"><div class=\"nav-row\"><label for=\"navbar-toggle\" role=\"button\" aria-label=\"menu\" class=\"navbar-burger\"><span aria-hidden=\"true\"></span><span aria-hidden=\"true\"></span><span aria-hidden=\"true\"></span></label><label for=\"nav-closed\" role=\"button\" class=\"navbar-item icon-button mobile-menu-close\" title=\"Close menu\" aria-label=\"Close menu\"><i class=\"fas fa-xmark fa-fw\"></i></label><div class=\"navbar-end\">")
	renderPreferenceLinksWithPaths(w, page, query, prefs, paths)
	renderSearchWithPaths(w, page, query, "search", "desktop-search", paths)
	write(w, "<label for=\"search-toggle\" role=\"button\" class=\"navbar-item icon-button mobile-search-toggle\" title=\"Search\" aria-label=\"Search\"><i class=\"fas fa-search fa-fw\"></i></label><label for=\"nav-closed\" role=\"button\" class=\"navbar-item icon-button mobile-search-close\" title=\"Close search\" aria-label=\"Close search\"><i class=\"fas fa-xmark fa-fw\"></i></label></div></div><div class=\"navbar-menu\"><div class=\"navbar-start\">")
	for _, link := range cfg.Links {
		write(w, "<a class=\"navbar-item\" rel=\"noreferrer\" href=\""+attr(navLinkURLWithPaths(link.URL, paths))+"\"")
		if link.Target != "" {
			write(w, " target=\""+attr(link.Target)+"\"")
		}
		write(w, ">")
		if link.Icon != "" {
			write(w, "<i class=\"fa-fw "+attr(link.Icon)+"\"></i>")
		}
		write(w, esc(link.Name)+"</a>")
	}
	write(w, "</div></div><div class=\"mobile-search-panel\">")
	renderSearchWithPaths(w, page, query, "mobile-search", "mobile-search", paths)
	write(w, "</div></div></nav></div>")
}

func renderSearch(w io.Writer, page, query, inputID, className string) {
	renderSearchWithPaths(w, page, query, inputID, className, Paths{})
}

func renderSearchWithPaths(w io.Writer, page, query, inputID, className string, paths Paths) {
	write(w, "<search class=\"search-bar "+attr(className)+"\"><form role=\"search\" method=\"get\" action=\""+attr(paths.URL("/"))+"\"><label for=\""+attr(inputID)+"\" class=\"search-label\"></label><input id=\""+attr(inputID)+"\" type=\"search\" name=\"search\" value=\""+attr(query)+"\"><button type=\"submit\" aria-label=\"Search\"><i class=\"fas fa-search\"></i></button>")
	if page != "" && page != "default" {
		write(w, "<input type=\"hidden\" name=\"page\" value=\""+attr(page)+"\">")
	}
	if query != "" {
		write(w, "<a class=\"icon-button\" href=\""+attr(pageURLWithPaths(page, nil, paths))+"\" title=\"Clear search\"><i class=\"fas fa-xmark\"></i></a>")
	}
	write(w, "</form></search>")
}

func renderPreferenceLinks(w io.Writer, page, query string, prefs Preferences) {
	renderPreferenceLinksWithPaths(w, page, query, prefs, Paths{})
}

func renderPreferenceLinksWithPaths(w io.Writer, page, query string, prefs Preferences, paths Paths) {
	nextTheme := map[string]string{"auto": "light", "light": "dark", "dark": "auto"}[prefs.Theme]
	if nextTheme == "" {
		nextTheme = "auto"
	}
	nextLayout := "list"
	if prefs.Layout == "list" {
		nextLayout = "columns"
	}
	returnTo := pageURLWithPaths(page, nil, paths)
	if query != "" {
		returnTo = pageURLWithPaths(page, url.Values{"search": []string{query}}, paths)
	}
	themeURL := paths.URL("/theme") + "?value=" + url.QueryEscape(nextTheme) + "&return=" + url.QueryEscape(returnTo)
	layoutURL := paths.URL("/layout") + "?value=" + url.QueryEscape(nextLayout) + "&return=" + url.QueryEscape(returnTo)
	write(w, "<a class=\"navbar-item is-inline-block-mobile icon-button refresh-button\" title=\"Refresh\" href=\""+attr(returnTo)+"\" aria-label=\"Refresh\"><i class=\"fa-solid fa-arrow-rotate-right fa-fw\"></i></a>")
	write(w, "<a class=\"navbar-item is-inline-block-mobile icon-button\" title=\"Theme\" href=\""+attr(themeURL)+"\"><i class=\""+attr(themeIcon(prefs.Theme))+" fa-fw\"></i></a>")
	write(w, "<a class=\"navbar-item is-inline-block-mobile icon-button\" title=\"Layout\" href=\""+attr(layoutURL)+"\"><i class=\""+attr(layoutIcon(prefs.Layout))+" fa-fw\"></i></a>")
}

func renderMessageFragment(w io.Writer, msg config.Message) {
	write(w, "<div data-message-fragment>")
	renderMessage(w, msg)
	write(w, "</div>")
}

func renderOfflineMessage(w io.Writer, cfg config.Config) {
	if !cfg.ConnectivityCheck {
		return
	}
	write(w, "<div class=\"offline-message mb-4\" data-offline-message role=\"alert\" aria-live=\"polite\" hidden>")
	write(w, "<i class=\"fa-solid fa-triangle-exclamation\"></i>")
	write(w, "<h1>Network unreachable <button type=\"button\" aria-label=\"Retry connection check\" class=\"retry-button\" data-offline-retry><i class=\"fas fa-redo-alt\"></i></button></h1>")
	write(w, "<p><a href=\"https://github.com/bastienwirtz/homer/blob/main/docs/configuration.md#connectivity-checks\">More information &rarr;</a></p>")
	write(w, "</div>")
}

func renderMessage(w io.Writer, msg config.Message) {
	if msg.Title == "" && msg.Content == "" {
		return
	}
	write(w, "<article class=\"message "+attr(msg.Style)+"\">")
	if msg.Title != "" || msg.Icon != "" {
		write(w, "<div class=\"message-header\"><p>")
		if msg.Icon != "" {
			write(w, "<i class=\"fa-fw "+attr(msg.Icon)+"\"></i>")
		}
		write(w, esc(msg.Title)+"</p></div>")
	}
	if msg.Content != "" {
		write(w, "<div class=\"message-body\">"+msg.Content+"</div>")
	}
	write(w, "</article>")
}

func renderServicesFragment(w io.Writer, cfg config.Config, prefs Preferences, statuses map[string]collectors.Status) {
	renderServicesFragmentWithPaths(w, cfg, prefs, statuses, Paths{})
}

func renderServicesFragmentWithPaths(w io.Writer, cfg config.Config, prefs Preferences, statuses map[string]collectors.Status, paths Paths) {
	write(w, "<div data-services-fragment>")
	renderGroupsWithPaths(w, cfg, prefs, statuses, paths)
	write(w, "</div>")
}

func renderGroups(w io.Writer, cfg config.Config, prefs Preferences, statuses map[string]collectors.Status) {
	renderGroupsWithPaths(w, cfg, prefs, statuses, Paths{})
}

func renderGroupsWithPaths(w io.Writer, cfg config.Config, prefs Preferences, statuses map[string]collectors.Status, paths Paths) {
	columns := columnsClass(cfg.Columns)
	write(w, "<section class=\"service-groups "+attr(layoutClass(prefs.Layout))+"\">")
	if !hasServiceItems(cfg) {
		write(w, "<p class=\"empty-state\">No results</p>")
		write(w, "</section>")
		return
	}
	if prefs.Layout == "list" {
		write(w, "<div class=\"columns is-multiline group-columns\">")
	}
	for groupIndex, group := range cfg.Services {
		groupClass := strings.TrimSpace("service-group " + group.Class)
		if prefs.Layout == "list" {
			groupClass = strings.TrimSpace("column " + bulmaColumnClass(cfg.Columns) + " " + groupClass)
		}
		write(w, "<section class=\""+attr(groupClass)+"\">")
		if group.Name != "" {
			write(w, "<h2 class=\"group-title "+attr(group.Class)+"\">")
			if group.Icon != "" {
				write(w, "<i class=\"fa-fw "+attr(group.Icon)+"\"></i>")
			} else if group.Logo != "" {
				write(w, "<div class=\"group-logo media-left\"><figure class=\"image is-48x48\"><img src=\""+attr(paths.AssetURL(group.Logo))+"\" alt=\""+attr(group.Name)+" logo\"></figure></div>")
			}
			write(w, esc(group.Name)+"</h2>")
		}
		write(w, "<div class=\"columns is-multiline items "+attr(columns)+"\">")
		for itemIndex, item := range group.Items {
			renderItemWithPaths(w, item, group.Class, statuses[collectors.Key(groupIndex, itemIndex)], prefs.Layout, cfg.Columns, paths)
		}
		write(w, "</div></section>")
	}
	if prefs.Layout == "list" {
		write(w, "</div>")
	}
	write(w, "</section>")
}

func hasServiceItems(cfg config.Config) bool {
	for _, group := range cfg.Services {
		if len(group.Items) > 0 {
			return true
		}
	}
	return false
}

func renderItem(w io.Writer, item config.Item, groupClass string, status collectors.Status, layout, columns string) {
	renderItemWithPaths(w, item, groupClass, status, layout, columns, Paths{})
}

func renderItemWithPaths(w io.Writer, item config.Item, groupClass string, status collectors.Status, layout, columns string, paths Paths) {
	columnClass := ""
	if layout != "list" {
		columnClass = "column " + bulmaColumnClass(columns)
	}
	itemClass := item.Class
	if itemClass == "" {
		itemClass = groupClass
	}
	classes := strings.TrimSpace(columnClass + " service-card-wrapper " + itemClass + " status-" + status.State)
	style := ""
	if item.Background != "" {
		style = " style=\"background-color:" + attr(item.Background) + "\""
	}
	write(w, "<div class=\""+attr(classes)+"\"><div class=\"card\""+style+"><a class=\"card-link\" href=\""+attr(cardURL(item, status))+"\" rel=\"noreferrer\"")
	if item.Target != "" {
		write(w, " target=\""+attr(item.Target)+"\"")
	}
	write(w, " aria-label=\""+attr(item.Name)+"\"></a>")
	mediaClass := "media"
	if item.Subtitle == "" && status.Label == "" {
		mediaClass += " no-subtitle"
	}
	write(w, "<div class=\"card-content\"><div class=\""+attr(mediaClass)+"\">")
	if item.Logo != "" {
		write(w, "<div class=\"media-left\"><figure class=\"image is-48x48\"><img src=\""+attr(paths.AssetURL(item.Logo))+"\" alt=\""+attr(item.Name)+" logo\"></figure></div>")
	} else if item.Icon != "" {
		write(w, "<div class=\"media-left\"><figure class=\"image is-48x48\"><i class=\"fa-fw "+attr(item.Icon)+"\"></i></figure></div>")
	}
	write(w, "<div class=\"media-content\"><p class=\"title\">"+esc(item.Name)+"</p>")
	renderQuickLinks(w, item)
	subtitle := item.Subtitle
	if subtitle == "" && status.Label != "" {
		subtitle = status.Label
	}
	if subtitle != "" {
		write(w, "<p class=\"subtitle\">"+esc(subtitle)+"</p>")
	}
	write(w, "</div>")
	renderStatusBadges(w, status)
	renderStatusIndicator(w, status)
	write(w, "</div></div>")
	if item.Tag != "" {
		write(w, "<div class=\"tag "+attr(item.TagStyle)+"\"><strong class=\"tag-text\">#"+esc(item.Tag)+"</strong></div>")
	}
	write(w, "</div></div>")
}

func cardURL(item config.Item, status collectors.Status) string {
	if status.URL != "" {
		return status.URL
	}
	return item.URL
}

func renderStatusBadges(w io.Writer, status collectors.Status) {
	if len(status.Badges) == 0 {
		return
	}
	write(w, "<div class=\"status-notifs\">")
	for _, badge := range status.Badges {
		classes := strings.TrimSpace("status-notif " + badge.State + " " + toneClass(badge.Tone))
		title := badge.Detail
		if title == "" {
			title = badge.Label
		}
		write(w, "<span class=\""+attr(classes)+"\"")
		if title != "" {
			write(w, " title=\""+attr(title)+"\"")
		}
		write(w, ">")
		if badge.Label != "" {
			write(w, "<span class=\"status-notif-label\">"+esc(badge.Label)+"</span>")
		}
		write(w, esc(badge.Value)+"</span>")
	}
	write(w, "</div>")
}

func renderStatusIndicator(w io.Writer, status collectors.Status) {
	if status.State == "" && status.Indicator == "" {
		return
	}
	textClass := ""
	if status.Indicator == "" {
		textClass = " is-dot"
	}
	classes := strings.TrimSpace("status-indicator" + textClass + " " + status.State + " " + toneClass(status.Tone))
	write(w, "<div class=\""+attr(classes)+"\"")
	if status.Detail != "" {
		write(w, " title=\""+attr(status.Detail)+"\"")
	}
	write(w, ">"+esc(status.Indicator)+"</div>")
}

func toneClass(tone string) string {
	normalized := normalizeTone(tone)
	switch normalized {
	case "success", "warning", "danger", "info", "neutral":
		return "is-" + normalized
	}
	return "is-neutral"
}

func normalizeTone(tone string) string {
	switch strings.ToLower(strings.TrimSpace(tone)) {
	case "success", "ok", "good":
		return "success"
	case "warning", "warn", "pending":
		return "warning"
	case "danger", "error", "bad":
		return "danger"
	case "info", "activity":
		return "info"
	case "neutral", "default", "unknown":
		return "neutral"
	default:
		return ""
	}
}

func renderQuickLinks(w io.Writer, item config.Item) {
	if len(item.Quick) == 0 {
		return
	}
	write(w, "<p class=\"quicklinks\">")
	for _, link := range item.Quick {
		write(w, "<a href=\""+attr(link.URL)+"\" rel=\"noreferrer\"")
		if link.Target != "" {
			write(w, " target=\""+attr(link.Target)+"\"")
		}
		if link.Color != "" {
			write(w, " style=\"background-color:"+attr(link.Color)+"\"")
		}
		write(w, ">")
		if link.Icon != "" {
			write(w, "<span><i class=\"fa-fw "+attr(link.Icon)+"\"></i></span>")
		}
		write(w, esc(link.Name)+"</a>")
	}
	write(w, "</p>")
}

func renderFooter(w io.Writer, cfg config.Config) {
	if footerHidden(cfg.Footer) {
		return
	}
	switch v := cfg.Footer.(type) {
	case string:
		write(w, "<footer class=\"footer\"><div class=\"container\"><div class=\"content has-text-centered\">"+v+"</div></div></footer>")
		return
	}
}

func footerHidden(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	default:
		return false
	}
}

func writeDynamicTheme(w io.Writer, cfg config.Config, paths Paths) {
	if len(cfg.Colors) == 0 {
		return
	}
	write(w, "<style>")
	for name, colors := range cfg.Colors {
		selector := "body #app." + name
		if name == "light" {
			selector = ":root, body #app.light"
		}
		write(w, selector+"{")
		for key, value := range colors {
			if key == "background-image" && value != "" {
				value = "url(" + paths.AssetURL(value) + ")"
			}
			write(w, "--"+key+":"+value+";")
		}
		write(w, "}")
	}
	write(w, "</style>")
}

func renderRuntimeConfig(w io.Writer, cfg config.Config, page string, paths Paths) {
	write(w, "<script>")
	write(w, "window.HOMER_UPDATE_INTERVAL_MS="+strconv.Itoa(clampInterval(cfg.UpdateIntervalMs))+";")
	write(w, "window.HOMER_MESSAGE_REFRESH_INTERVAL="+strconv.Itoa(clampInterval(cfg.Message.RefreshInterval))+";")
	write(w, "window.HOMER_PAGE="+strconv.Quote(page)+";")
	write(w, "window.HOMER_BASE_PATH="+strconv.Quote(paths.BasePath)+";")
	write(w, "window.HOMER_CONNECTIVITY_CHECK="+strconv.FormatBool(cfg.ConnectivityCheck)+";")
	write(w, "</script>")
	write(w, "<script>if(\"serviceWorker\" in navigator){navigator.serviceWorker.register("+strconv.Quote(paths.URL("/sw.js"))+");let controlled=!!navigator.serviceWorker.controller;let refreshing=false;navigator.serviceWorker.addEventListener(\"controllerchange\",function(){if(!controlled){controlled=true;return;}if(refreshing)return;refreshing=true;window.location.reload();});}</script>")
	write(w, "<script src=\""+attr(paths.URL("/assets/homer-go.js"))+"\" defer></script>")
}

func clampInterval(interval int) int {
	if interval <= 0 {
		return 0
	}
	if interval < 1000 {
		return 1000
	}
	return interval
}

func themeClass(theme string) string {
	switch theme {
	case "light", "dark":
		return theme
	default:
		return "auto"
	}
}

func themeIcon(theme string) string {
	switch theme {
	case "light":
		return "fas fa-circle"
	case "dark":
		return "far fa-circle"
	default:
		return "fas fa-adjust"
	}
}

func layoutIcon(layout string) string {
	if layout == "list" {
		return "fas fa-columns"
	}
	return "fas fa-list"
}

func layoutClass(layout string) string {
	if layout == "list" {
		return "layout-list layout-vertical"
	}
	return "layout-columns"
}

func navLinkURL(raw string) string {
	return navLinkURLWithPaths(raw, Paths{})
}

func navLinkURLWithPaths(raw string, paths Paths) string {
	if raw == "#" {
		return raw
	}
	page, ok := pageFromHashURL(raw)
	if !ok {
		return raw
	}
	return pageURLWithPaths(page, nil, paths)
}

func pageFromHashURL(raw string) (string, bool) {
	if !strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "##") {
		return "", false
	}
	page := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	if page == "" {
		return "", false
	}
	for _, r := range page {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return "", false
	}
	return page, true
}

func pageURL(page string, params url.Values) string {
	return pageURLWithPaths(page, params, Paths{})
}

func pageURLWithPaths(page string, params url.Values, paths Paths) string {
	values := url.Values{}
	if page != "" && page != "default" {
		values.Set("page", page)
	}
	for key, items := range params {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	if len(values) == 0 {
		return paths.URL("/")
	}
	return paths.URL("/") + "?" + values.Encode()
}

func pageClass(page string) string {
	if page == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range page {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func columnsClass(columns string) string {
	if columns == "auto" {
		return "cols-auto"
	}
	n, err := strconv.Atoi(columns)
	if err != nil || n <= 0 {
		return "cols-3"
	}
	return "cols-" + strconv.Itoa(n)
}

func bulmaColumnClass(columns string) string {
	if columns == "auto" {
		return ""
	}
	n, err := strconv.Atoi(columns)
	if err != nil || n <= 0 {
		n = 3
	}
	width := 12 / n
	if width <= 0 {
		width = 4
	}
	return "is-" + strconv.Itoa(width)
}

func esc(s string) string {
	return html.EscapeString(s)
}

func attr(s string) string {
	return html.EscapeString(s)
}

func write(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}
