package collectors

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"time"

	"homer-go/internal/config"
)

func deviceTotalsStatus(total, connected, newDevices, down int) Status {
	return Status{
		Badges: positiveBadges(
			countBadge("Total", total, "total", "info"),
			countBadge("Connected", connected, "connected", "success"),
			countBadge("New", newDevices, "new", "warning"),
			countBadge("Down", down, "down", "danger"),
		),
		Updated: time.Now(),
	}
}

func totalAt(values []int, index int) int {
	if index >= 0 && index < len(values) {
		return values[index]
	}
	return 0
}

func netAlertTotals(value any) (int, int, int, int) {
	switch v := value.(type) {
	case []any:
		return intAtAny(v, 0), intAtAny(v, 1), intAtAny(v, 3), intAtAny(v, 4)
	case map[string]any:
		return asCollectorInt(v["total"]), asCollectorInt(v["connected"]), asCollectorInt(v["new"]), asCollectorInt(v["down"])
	default:
		return 0, 0, 0, 0
	}
}

func intAtAny(values []any, index int) int {
	if index >= 0 && index < len(values) {
		return asCollectorInt(values[index])
	}
	return 0
}

func peanutState(code string) (string, string, string) {
	switch code {
	case "OL":
		return "online", "success", "online"
	case "OB":
		return "pending", "warning", "on battery"
	case "LB":
		return "offline", "danger", "low battery"
	default:
		return "unknown", "warning", "unknown"
	}
}

func gatusState(up, down, total int) (string, string) {
	switch {
	case up == 0 && down == 0:
		return "unknown", "neutral"
	case down == total:
		return "bad", "danger"
	case up == total:
		return "good", "success"
	default:
		return "warn", "warning"
	}
}

func humanizeBytes(value int64) string {
	bytes := float64(value)
	if math.Abs(bytes) < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	unit := -1
	for math.Round(math.Abs(bytes)*100)/100 >= 1024 && unit < len(units)-1 {
		bytes /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", bytes, units[unit])
}

func humanizeRate(value float64, units []string) string {
	unit := 0
	for value > 1000 && unit < len(units)-1 {
		value /= 1000
		unit++
	}
	return fmt.Sprintf("%.2f %s/s", value, units[unit])
}

func formatMetric(value float64) string {
	if value == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", value)
}

func basicAuth(value string) string {
	if value == "" {
		return ""
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(value))
}

func asCollectorInt(value any) int {
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

func asCollectorFloat(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	default:
		return 0
	}
}

func asCollectorString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func boolField(item config.Item, name string) bool {
	value, ok := item.Raw[name].(bool)
	return ok && value
}

func stringSliceField(item config.Item, name string) []string {
	value, ok := item.Raw[name]
	if !ok {
		return nil
	}
	if s, ok := value.(string); ok && s != "" {
		return []string{s}
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstStringField(item config.Item, names ...string) string {
	for _, name := range names {
		if value := stringField(item, name); value != "" {
			return value
		}
	}
	return ""
}

func stringSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}
