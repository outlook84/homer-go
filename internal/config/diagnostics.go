package config

import "sort"

func UnsupportedConfigPaths(cfg Config) []string {
	var paths []string

	if hasRawKey(cfg.Raw, "hotkey") {
		paths = append(paths, "hotkey")
	}

	proxy := asMap(cfg.Raw["proxy"])
	if hasRawKey(proxy, "useCredentials") {
		paths = append(paths, "proxy.useCredentials")
	}

	for groupIndex, group := range cfg.Services {
		for itemIndex, item := range group.Items {
			prefix := "services[" + itoa(groupIndex) + "].items[" + itoa(itemIndex) + "]"
			if hasRawKey(item.Raw, "useCredentials") {
				paths = append(paths, prefix+".useCredentials")
			}
			for _, key := range itemUpdateIntervalKeys() {
				if hasRawKey(item.Raw, key) {
					paths = append(paths, prefix+"."+key)
				}
			}
			if item.Type == "Ping" && hasRawKey(item.Raw, "timeout") {
				paths = append(paths, prefix+".timeout")
			}
		}
	}

	sort.Strings(paths)
	return paths
}

func itemUpdateIntervalKeys() []string {
	return []string{
		"updateIntervalMs",
		"checkInterval",
		"downloadInterval",
		"rateInterval",
		"torrentInterval",
		"updateInterval",
	}
}

func hasRawKey(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
