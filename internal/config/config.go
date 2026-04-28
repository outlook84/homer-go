package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	AssetsDir     string
	ConfigDir     string
	ExampleConfig []byte
	AutoInit      bool
	OnInit        func(string)
}

type ErrorKind string

const (
	ErrorNotFound     ErrorKind = "not_found"
	ErrorPageNotFound ErrorKind = "page_not_found"
	ErrorConfigDir    ErrorKind = "config_dir"
	ErrorParse        ErrorKind = "parse"
	ErrorExternal     ErrorKind = "external"
)

type LoadError struct {
	Kind   ErrorKind
	Source string
	Err    error
}

func (e *LoadError) Error() string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case ErrorNotFound:
		return fmt.Sprintf("configuration %s was not found", e.Source)
	case ErrorPageNotFound:
		return fmt.Sprintf("page configuration %s was not found", e.Source)
	case ErrorConfigDir:
		return fmt.Sprintf("configuration directory %s is not available: %v", e.Source, e.Err)
	case ErrorParse:
		return fmt.Sprintf("configuration %s could not be parsed: %v", e.Source, e.Err)
	case ErrorExternal:
		return fmt.Sprintf("external configuration %s could not be loaded: %v", e.Source, e.Err)
	default:
		return e.Err.Error()
	}
}

func (e *LoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Config struct {
	Raw               map[string]any
	Title             string
	Subtitle          string
	DocumentTitle     string
	Logo              string
	Icon              string
	Header            bool
	Footer            any
	Columns           string
	ConnectivityCheck bool
	Defaults          Defaults
	Theme             string
	Colors            map[string]map[string]string
	Stylesheet        []string
	Message           Message
	Links             []Link
	Services          []Group
	Proxy             Proxy
	UpdateIntervalMs  int
}

type Proxy struct {
	UseCredentials bool
	Headers        map[string]string
}

type Defaults struct {
	Layout     string
	ColorTheme string
}

type Message struct {
	URL             string
	Mapping         map[string]string
	RefreshInterval int
	Style           string
	Title           string
	Icon            string
	Content         string
}

type Link struct {
	Name   string
	Icon   string
	URL    string
	Target string
}

type Group struct {
	Name     string
	Icon     string
	Logo     string
	Class    string
	TagStyle string
	Items    []Item
	Raw      map[string]any
}

type Item struct {
	Name       string
	Logo       string
	Icon       string
	Subtitle   string
	Tag        string
	Keywords   string
	URL        string
	Target     string
	TagStyle   string
	Type       string
	Class      string
	Quick      []QuickLink
	Background string
	Headers    map[string]string
	Raw        map[string]any
}

type QuickLink struct {
	Name   string
	Icon   string
	URL    string
	Target string
	Color  string
}

func (l Loader) Load(ctx context.Context, page string) (Config, error) {
	page = strings.TrimSpace(page)
	if page == "" {
		page = "default"
	}
	if page != "default" && !ValidPageName(page) {
		return Config{}, &LoadError{Kind: ErrorPageNotFound, Source: page, Err: fmt.Errorf("invalid page name")}
	}

	base, err := l.loadBaseMap(ctx)
	if err != nil {
		return Config{}, err
	}
	if page != "" && page != "default" {
		pageMap, err := l.loadPageMap(ctx, page)
		if err != nil {
			var loadErr *LoadError
			if errors.As(err, &loadErr) && loadErr.Kind == ErrorNotFound {
				loadErr.Kind = ErrorPageNotFound
			}
			return Config{}, err
		}
		for key, value := range pageMap {
			base[key] = value
		}
	}
	merged := deepMerge(defaultMap(), base)
	cfg, err := decode(merged)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ResolveMessage(ctx context.Context, msg Message) Message {
	return resolveRemoteMessage(ctx, msg)
}

func (l Loader) loadBaseMap(ctx context.Context) (map[string]any, error) {
	path := l.baseConfigPath()
	m, err := l.loadMap(ctx, path)
	if err == nil {
		return m, nil
	}
	var loadErr *LoadError
	if !l.shouldInitConfig() || !errors.As(err, &loadErr) || loadErr.Kind != ErrorNotFound {
		return nil, err
	}
	if err := requireConfigDir(filepath.Dir(path)); err != nil {
		return nil, &LoadError{Kind: ErrorConfigDir, Source: filepath.Dir(path), Err: err}
	}
	if writeErr := os.WriteFile(path, l.ExampleConfig, 0o600); writeErr != nil {
		return nil, &LoadError{Kind: ErrorNotFound, Source: path, Err: writeErr}
	}
	if l.OnInit != nil {
		l.OnInit(path)
	}
	return l.loadMap(ctx, path)
}

func (l Loader) loadPageMap(ctx context.Context, page string) (map[string]any, error) {
	if !ValidPageName(page) {
		return nil, &LoadError{Kind: ErrorPageNotFound, Source: page, Err: fmt.Errorf("invalid page name")}
	}
	return l.loadMap(ctx, filepath.Join(filepath.Dir(l.baseConfigPath()), page+".yml"))
}

func (l Loader) loadMap(ctx context.Context, path string) (map[string]any, error) {
	m, err := readYAML(ctx, path, false)
	if err != nil {
		return nil, err
	}
	if external, ok := stringValue(m["externalConfig"]); ok && external != "" {
		m, err := readYAML(ctx, external, true)
		if err != nil {
			var loadErr *LoadError
			if errors.As(err, &loadErr) && loadErr.Kind == ErrorExternal {
				return nil, err
			}
			return nil, &LoadError{Kind: ErrorExternal, Source: external, Err: err}
		}
		return m, nil
	}
	return m, nil
}

func (l Loader) baseConfigPath() string {
	dir := l.ConfigDir
	if dir == "" {
		dir = l.AssetsDir
	}
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "config.yml")
}

func (l Loader) shouldInitConfig() bool {
	return l.AutoInit && len(l.ExampleConfig) > 0
}

func ValidPageName(page string) bool {
	if page == "" {
		return false
	}
	for _, r := range page {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func requireConfigDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("create the directory first or choose an existing directory")
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func readYAML(ctx context.Context, path string, external bool) (map[string]any, error) {
	var body []byte
	var err error
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		if reqErr != nil {
			return nil, wrapReadError(path, reqErr, external)
		}
		resp, reqErr := http.DefaultClient.Do(req)
		if reqErr != nil {
			return nil, wrapReadError(path, reqErr, external)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, wrapReadError(path, fmt.Errorf("load %s: %s", path, resp.Status), external)
		}
		body, err = ioReadAll(resp.Body)
	} else {
		body, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, wrapReadError(path, err, external)
	}
	var out map[string]any
	if err := yaml.Unmarshal(body, &out); err != nil {
		return nil, &LoadError{Kind: errorKind(ErrorParse, external), Source: path, Err: err}
	}
	if out == nil {
		out = map[string]any{}
	}
	return normalizeMap(out), nil
}

func wrapReadError(path string, err error, external bool) error {
	if external {
		return &LoadError{Kind: ErrorExternal, Source: path, Err: err}
	}
	if errors.Is(err, os.ErrNotExist) {
		return &LoadError{Kind: ErrorNotFound, Source: path, Err: err}
	}
	return err
}

func errorKind(kind ErrorKind, external bool) ErrorKind {
	if external {
		return ErrorExternal
	}
	return kind
}

func decode(raw map[string]any) (Config, error) {
	cfg := Config{
		Raw:               raw,
		Title:             asString(raw["title"]),
		Subtitle:          asString(raw["subtitle"]),
		DocumentTitle:     asString(raw["documentTitle"]),
		Logo:              asString(raw["logo"]),
		Icon:              asString(raw["icon"]),
		Header:            asBoolDefault(raw["header"], true),
		Footer:            raw["footer"],
		Columns:           asColumns(raw["columns"]),
		ConnectivityCheck: asBoolDefault(raw["connectivityCheck"], true),
		Theme:             asStringDefault(raw["theme"], "default"),
		Stylesheet:        asStringSlice(raw["stylesheet"]),
		Proxy:             asProxy(raw["proxy"]),
		UpdateIntervalMs:  asInt(raw["updateIntervalMs"]),
	}
	defaults := asMap(raw["defaults"])
	cfg.Defaults = Defaults{
		Layout:     asStringDefault(defaults["layout"], "columns"),
		ColorTheme: asStringDefault(defaults["colorTheme"], "auto"),
	}
	cfg.Colors = asColorSets(raw["colors"])
	cfg.Message = asMessage(raw["message"])
	cfg.Links = asLinks(raw["links"])
	cfg.Services = asGroups(raw["services"])
	return cfg, nil
}

func resolveRemoteMessage(ctx context.Context, msg Message) Message {
	if msg.URL == "" {
		return msg
	}

	ctx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, msg.URL, nil)
	if err != nil {
		return msg
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return msg
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return msg
	}

	var remote map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&remote); err != nil {
		return msg
	}
	if msg.Mapping != nil {
		remote = mapRemoteMessage(remote, msg.Mapping)
	}
	applyRemoteMessage(&msg, remote)
	return msg
}

func mapRemoteMessage(remote map[string]any, mapping map[string]string) map[string]any {
	mapped := map[string]any{}
	for target, source := range mapping {
		if value, ok := remote[source]; ok && remoteTruthy(value) {
			mapped[target] = value
		}
	}
	return mapped
}

func applyRemoteMessage(msg *Message, remote map[string]any) {
	for _, field := range []string{"title", "style", "content", "icon"} {
		value, ok := remote[field]
		if !ok || value == nil {
			continue
		}
		text := remoteString(value)
		switch field {
		case "title":
			msg.Title = text
		case "style":
			msg.Style = text
		case "content":
			msg.Content = text
		case "icon":
			msg.Icon = text
		}
	}
}

func remoteTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case bool:
		return v
	case float64:
		return v != 0
	default:
		return true
	}
}

func remoteString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		body, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(body)
	}
}

func (c Config) Filter(query string) Config {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return c
	}
	next := c
	next.Services = []Group{{
		Name:  query,
		Icon:  "fas fa-search",
		Items: nil,
	}}
	for _, group := range c.Services {
		for _, item := range group.Items {
			if item.matches(query) {
				next.Services[0].Items = append(next.Services[0].Items, item)
			}
		}
	}
	return next
}

func (i Item) matches(query string) bool {
	values := []string{i.Name, i.Subtitle, i.Tag, i.Keywords}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
