package config

import (
	"io"
	"strconv"
)

func ioReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func defaultMap() map[string]any {
	return map[string]any{
		"title":             "Dashboard",
		"subtitle":          "Homer",
		"header":            true,
		"footer":            `<p>Created with <span class="has-text-danger">heart</span> with <a href="https://bulma.io/">bulma</a>, <a href="https://vuejs.org/">vuejs</a> & <a href="https://fontawesome.com/">font awesome</a></p>`,
		"columns":           "3",
		"connectivityCheck": true,
		"defaults": map[string]any{
			"layout":     "columns",
			"colorTheme": "auto",
		},
		"theme":    "default",
		"colors":   nil,
		"message":  nil,
		"links":    []any{},
		"services": []any{},
		"proxy":    nil,
	}
}

func deepMerge(dst, src map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range dst {
		out[key] = value
	}
	for key, value := range src {
		left, leftOK := out[key].(map[string]any)
		right, rightOK := value.(map[string]any)
		if leftOK && rightOK {
			out[key] = deepMerge(left, right)
			continue
		}
		out[key] = value
	}
	return out
}

func normalizeMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = normalizeValue(value)
	}
	return out
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return normalizeMap(v)
	case []any:
		for i := range v {
			v[i] = normalizeValue(v[i])
		}
		return v
	default:
		return value
	}
}

func asMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func asStringDefault(value any, fallback string) string {
	if s := asString(value); s != "" {
		return s
	}
	return fallback
}

func asColumns(value any) string {
	columns := asStringDefault(value, "3")
	switch columns {
	case "auto", "1", "2", "3", "4", "6", "12":
		return columns
	default:
		return "3"
	}
}

func asBoolDefault(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return fallback
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func stringValue(value any) (string, bool) {
	s := asString(value)
	return s, s != ""
}

func asStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	if s := asString(value); s != "" {
		return []string{s}
	}
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := asString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func asColorSets(value any) map[string]map[string]string {
	sets := map[string]map[string]string{}
	for name, rawSet := range asMap(value) {
		set := map[string]string{}
		for key, rawValue := range asMap(rawSet) {
			set[key] = asString(rawValue)
		}
		sets[name] = set
	}
	if len(sets) == 0 {
		return nil
	}
	return sets
}

func asMessage(value any) Message {
	m := asMap(value)
	return Message{
		URL:             asString(m["url"]),
		Mapping:         asStringMap(m["mapping"]),
		RefreshInterval: asInt(m["refreshInterval"]),
		Style:           asString(m["style"]),
		Title:           asString(m["title"]),
		Icon:            asString(m["icon"]),
		Content:         asString(m["content"]),
	}
}

func asProxy(value any) Proxy {
	m := asMap(value)
	return Proxy{
		UseCredentials: asBoolDefault(m["useCredentials"], false),
		Headers:        asStringMap(m["headers"]),
	}
}

func asStringMap(value any) map[string]string {
	m := asMap(value)
	if len(m) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range m {
		out[key] = asString(value)
	}
	return out
}

func asLinks(value any) []Link {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]Link, 0, len(arr))
	for _, raw := range arr {
		m := asMap(raw)
		out = append(out, Link{
			Name:   asString(m["name"]),
			Icon:   asString(m["icon"]),
			URL:    asString(m["url"]),
			Target: asString(m["target"]),
		})
	}
	return out
}

func asGroups(value any) []Group {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]Group, 0, len(arr))
	for _, raw := range arr {
		m := asMap(raw)
		out = append(out, Group{
			Name:     asString(m["name"]),
			Icon:     asString(m["icon"]),
			Logo:     asString(m["logo"]),
			Class:    asString(m["class"]),
			TagStyle: asString(m["tagstyle"]),
			Items:    asItems(m["items"]),
			Raw:      m,
		})
	}
	return out
}

func asItems(value any) []Item {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]Item, 0, len(arr))
	for _, raw := range arr {
		m := asMap(raw)
		out = append(out, Item{
			Name:       asString(m["name"]),
			Logo:       asString(m["logo"]),
			Icon:       asString(m["icon"]),
			Subtitle:   asString(m["subtitle"]),
			Tag:        asString(m["tag"]),
			Keywords:   asString(m["keywords"]),
			URL:        asString(m["url"]),
			Target:     asString(m["target"]),
			TagStyle:   asString(m["tagstyle"]),
			Type:       asStringDefault(m["type"], "Generic"),
			Class:      asString(m["class"]),
			Quick:      asQuickLinks(m["quick"]),
			Background: asString(m["background"]),
			Headers:    asStringMap(m["headers"]),
			Raw:        m,
		})
	}
	return out
}

func asQuickLinks(value any) []QuickLink {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]QuickLink, 0, len(arr))
	for _, raw := range arr {
		m := asMap(raw)
		out = append(out, QuickLink{
			Name:   asString(m["name"]),
			Icon:   asString(m["icon"]),
			URL:    asString(m["url"]),
			Target: asString(m["target"]),
			Color:  asString(m["color"]),
		})
	}
	return out
}
