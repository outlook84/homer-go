package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type WUD struct{}

func (WUD) Type() string { return "WUD" }

func (WUD) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var containers []struct {
		UpdateAvailable bool `json:"updateAvailable"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/containers"}, &containers); err != nil {
		return offlineStatus("Error", err)
	}
	running := len(containers)
	updates := 0
	for _, container := range containers {
		if container.UpdateAvailable {
			updates++
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Running", running, "running", "warning"),
			countBadge("Update", updates, "update", "danger"),
		),
		Updated: time.Now(),
	}
}
