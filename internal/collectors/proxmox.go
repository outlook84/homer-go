package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"homer-go/internal/config"
)

type Proxmox struct{}

func (Proxmox) Type() string { return "Proxmox" }

func (Proxmox) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if token := proxmoxAuthorization(item); token != "" {
		headers["Authorization"] = token
	}
	node := stringField(item, "node")
	var nodeStatus struct {
		Data struct {
			Memory struct {
				Used  float64 `json:"used"`
				Total float64 `json:"total"`
			} `json:"memory"`
			RootFS struct {
				Used  float64 `json:"used"`
				Total float64 `json:"total"`
			} `json:"rootfs"`
			CPU float64 `json:"cpu"`
		} `json:"data"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api2/json/nodes/" + node + "/status", Headers: headers}, &nodeStatus); err != nil {
		return offlineStatus("Error", err)
	}
	decimals := 1
	if boolField(item, "hide_decimals") {
		decimals = 0
	}
	hide := stringSet(stringSliceField(item, "hide"))
	badges := []Badge{
		percentBadge("Disk", percent(nodeStatus.Data.RootFS.Used, nodeStatus.Data.RootFS.Total), decimals, hide, item),
		percentBadge("Mem", percent(nodeStatus.Data.Memory.Used, nodeStatus.Data.Memory.Total), decimals, hide, item),
		percentBadge("CPU", nodeStatus.Data.CPU*100, decimals, hide, item),
	}
	if !hide["vms"] {
		total, running, err := proxmoxResourceCounts(ctx, item, proxy, "api2/json/nodes/"+node+"/qemu", headers)
		if err != nil {
			return offlineStatus("Error", err)
		}
		badges = append(badges, Badge{Label: "VMs", Value: proxmoxCountValue(running, total, hide["vms_total"]), State: "vms", Tone: "info"})
	}
	if !hide["lxcs"] {
		total, running, err := proxmoxResourceCounts(ctx, item, proxy, "api2/json/nodes/"+node+"/lxc", headers)
		if err != nil {
			return offlineStatus("Error", err)
		}
		badges = append(badges, Badge{Label: "LXCs", Value: proxmoxCountValue(running, total, hide["lxcs_total"]), State: "lxcs", Tone: "neutral"})
	}
	return Status{Badges: compactBadges(badges), Updated: time.Now()}
}

func proxmoxAuthorization(item config.Item) string {
	if token := stringField(item, "api_token"); token != "" {
		if strings.HasPrefix(token, "PVEAPIToken=") {
			return token
		}
		if tokenID := firstStringField(item, "api_token_id", "token_id"); tokenID != "" {
			return "PVEAPIToken=" + tokenID + "=" + token
		}
		return token
	}
	tokenID := firstStringField(item, "api_token_id", "token_id")
	secret := firstStringField(item, "api_token_secret", "token_secret")
	if tokenID != "" && secret != "" {
		return "PVEAPIToken=" + tokenID + "=" + secret
	}
	return ""
}

func proxmoxResourceCounts(ctx context.Context, item config.Item, proxy config.Proxy, path string, headers map[string]string) (int, int, error) {
	var response struct {
		Data []struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: path, Headers: headers}, &response); err != nil {
		return 0, 0, err
	}
	running := 0
	for _, resource := range response.Data {
		if resource.Status == "running" {
			running++
		}
	}
	return len(response.Data), running, nil
}

func percent(used, total float64) float64 {
	if total == 0 {
		return 0
	}
	return used * 100 / total
}

func percentBadge(label string, value float64, decimals int, hide map[string]bool, item config.Item) Badge {
	key := strings.ToLower(label)
	if hide[key] {
		return Badge{}
	}
	format := "%.1f%%"
	if decimals == 0 {
		format = "%.0f%%"
	}
	return Badge{Label: label, Value: fmt.Sprintf(format, value), State: key, Tone: proxmoxTone(value, item)}
}

func proxmoxTone(value float64, item config.Item) string {
	danger := asCollectorFloat(item.Raw["danger_value"])
	warning := asCollectorFloat(item.Raw["warning_value"])
	switch {
	case danger > 0 && value > danger:
		return "danger"
	case warning > 0 && value > warning:
		return "warning"
	default:
		return "info"
	}
}

func proxmoxCountValue(running, total int, hideTotal bool) string {
	if hideTotal {
		return fmt.Sprintf("%d", running)
	}
	return fmt.Sprintf("%d/%d", running, total)
}

func compactBadges(badges []Badge) []Badge {
	out := []Badge{}
	for _, badge := range badges {
		if badge.Label != "" && badge.Value != "" {
			out = append(out, badge)
		}
	}
	return out
}
