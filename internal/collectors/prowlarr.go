package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Prowlarr struct{}

func (Prowlarr) Type() string { return "Prowlarr" }

func (Prowlarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var health []struct {
		Type string `json:"type"`
	}
	path := "api/v1/health?apikey=" + stringField(item, "apikey")
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: path}, &health); err != nil {
		return offlineStatus("Error", err)
	}
	warnings := 0
	errors := 0
	for _, item := range health {
		switch item.Type {
		case "warning":
			warnings++
		case "error":
			errors++
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Warning", warnings, "warnings", "warning"),
			countBadge("Error", errors, "errors", "danger"),
		),
		Updated: time.Now(),
	}
}
