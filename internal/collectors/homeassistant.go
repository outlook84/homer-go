package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"homer-go/internal/config"
)

type HomeAssistant struct{}

func (HomeAssistant) Type() string { return "HomeAssistant" }

func (HomeAssistant) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{"Content-Type": "application/json"}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	var root struct {
		Message string `json:"message"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/", Headers: headers}, &root); err != nil {
		return offlineStatus("dead", err)
	}
	if root.Message == "" {
		return offlineStatus("dead", fmt.Errorf("missing API message"))
	}
	status := onlineStatus("running", "")
	status.State = "running"
	status.Indicator = "running"
	if item.Subtitle != "" {
		return status
	}
	var cfg struct {
		Version      string `json:"version"`
		LocationName string `json:"location_name"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/config", Headers: headers}, &cfg); err != nil {
		return offlineStatus("dead", err)
	}
	var states []json.RawMessage
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/states", Headers: headers}, &states); err != nil {
		return offlineStatus("dead", err)
	}
	keys := stringSliceField(item, "items")
	if len(keys) == 0 {
		keys = []string{"name", "version"}
	}
	parts := []string{}
	for _, key := range keys {
		switch key {
		case "name":
			parts = append(parts, cfg.LocationName)
		case "version":
			parts = append(parts, "v"+cfg.Version)
		case "entities":
			parts = append(parts, fmt.Sprintf("%d entities", len(states)))
		}
	}
	separator := stringField(item, "separator")
	if separator == "" {
		separator = " "
	}
	status.Label = strings.Join(parts, separator)
	return status
}
