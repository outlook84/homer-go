package collectors

import (
	"context"
	"net/url"

	"homer-go/internal/config"
)

type HyperHDR struct{}

func (HyperHDR) Type() string { return "HyperHDR" }

func (HyperHDR) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	command := url.QueryEscape(`{"command":"serverinfo"}`)
	var response struct {
		Info struct {
			CurrentInstance int `json:"currentInstance"`
			Instance        []struct {
				Instance     int    `json:"instance"`
				FriendlyName string `json:"friendly_name"`
				Running      bool   `json:"running"`
			} `json:"instance"`
		} `json:"info"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "json-rpc?request=" + command}, &response); err != nil {
		return offlineStatus("offline", err)
	}
	running := 0
	current := ""
	for _, instance := range response.Info.Instance {
		if instance.Running {
			running++
		}
		if instance.Instance == response.Info.CurrentInstance {
			current = instance.FriendlyName
		}
	}
	status := onlineStatus("", "")
	status.State = "online"
	status.Indicator = "online"
	if current != "" {
		status.Label = "Current instance: " + current
	}
	status.Badges = positiveBadges(
		countBadge("Running", running, "running", "success"),
		countBadge("Stopped", len(response.Info.Instance)-running, "stopped", "danger"),
	)
	return status
}
