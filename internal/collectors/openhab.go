package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"homer-go/internal/config"
)

type OpenHAB struct{}

func (OpenHAB) Type() string { return "OpenHAB" }

func (OpenHAB) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["Authorization"] = basicAuth(apiKey + ":")
	}
	var system struct {
		SystemInfo any `json:"systemInfo"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "rest/systeminfo", Headers: headers}, &system); err != nil {
		return offlineStatus("dead", err)
	}
	if system.SystemInfo == nil {
		return offlineStatus("dead", fmt.Errorf("missing system info"))
	}
	status := onlineStatus("running", "")
	status.State = "running"
	status.Indicator = "running"
	if item.Subtitle != "" {
		return status
	}
	parts := []string{}
	if boolField(item, "things") {
		var things []struct {
			StatusInfo struct {
				Status string `json:"status"`
			} `json:"statusInfo"`
		}
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: "rest/things?summary=true", Headers: headers}, &things); err != nil {
			return offlineStatus("dead", err)
		}
		online := 0
		for _, thing := range things {
			if thing.StatusInfo.Status == "ONLINE" {
				online++
			}
		}
		parts = append(parts, fmt.Sprintf("%d things (%d Online)", len(things), online))
	}
	if boolField(item, "items") {
		var items []json.RawMessage
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: "rest/items", Headers: headers}, &items); err != nil {
			return offlineStatus("dead", err)
		}
		parts = append(parts, fmt.Sprintf("%d items", len(items)))
	}
	status.Label = strings.Join(parts, ", ")
	return status
}
